package mempool

import (
	"bytes"
	"encoding/hex"
	"errors"
	"math"
	"time"

	logger "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/heavy"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/verifiers"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

var log *logger.Entry = logger.WithFields(logger.Fields{"prefix": "mempool"})

const (
	consensusSeconds = 20
	maxPendingLen    = 1000
)

// Mempool is a storage for the chain transactions that are valid according to the
// current chain state and can be included in the next block.
type Mempool struct {

	// transactions emitted by RPC and Peer subsystems
	// pending to be verified before adding them to verified pool
	pending chan TxDesc

	// verified txs to be included in next block
	verified Pool

	// the collector to listen for new accepted blocks
	accepted Collector

	// used by tx verification procedure
	latestBlockTimestamp int64

	eventBus *wire.EventBus
	db       database.DB

	// the magic function that knows best what is valid chain Tx
	verifyTx func(tx transactions.Transaction) error
	quitChan chan struct{}
}

// checkTx is responsible to determine if a tx is valid or not
func (m *Mempool) checkTx(tx transactions.Transaction) error {

	// check if external verifyTx is provided
	if m.verifyTx != nil {
		return m.verifyTx(tx)
	}

	// retrieve read-only connection to the blockchain database
	if m.db == nil {
		_, m.db = heavy.CreateDBConnection()
	}

	// run the default blockchain verifier
	approxBlockTime := uint64(consensusSeconds) + uint64(m.latestBlockTimestamp)
	return verifiers.CheckTx(m.db, 0, approxBlockTime, tx)
}

type Collector struct {
	blockChan chan block.Block
}

func (c *Collector) Collect(msg *bytes.Buffer) error {
	b := block.NewBlock()
	if err := b.Decode(msg); err != nil {
		return err
	}

	c.blockChan <- *b
	return nil
}

// NewMempool instantiates and initializes node mempool
func NewMempool(eventBus *wire.EventBus, verifyTx func(tx transactions.Transaction) error) *Mempool {

	log.Infof("create new instance")

	m := &Mempool{
		eventBus:             eventBus,
		latestBlockTimestamp: math.MinInt32,
		quitChan:             make(chan struct{})}

	if verifyTx != nil {
		m.verifyTx = verifyTx
	}

	m.verified = m.newPool()

	log.Infof("running with pool type %s", config.Get().Mempool.PoolType)

	// topics.Tx will be published by RPC subsystem or Peer subsystem (deserialized from gossip msg)
	m.pending = make(chan TxDesc, maxPendingLen)
	go wire.NewTopicListener(m.eventBus, m, string(topics.Tx)).Accept()

	// topics.AcceptedBlock will be published by Chain subsystem when new block is accepted into blockchain
	m.accepted.blockChan = make(chan block.Block)
	go wire.NewTopicListener(m.eventBus, &m.accepted, string(topics.AcceptedBlock)).Accept()

	return m
}

// Run spawns the mempool lifecycle routine. The whole mempool cycle is around
// getting input from the outside world (from input channels) and provide the
// actual list of the verified txs (onto output channel).
//
// All operations are always executed in a single go-routine so no
// protection-by-mutex needed
func (m *Mempool) Run() {
	go func() {
		for {
			select {
			case r := <-wire.GetMempoolTxsChan:
				m.onGetMempoolTxs(r)
			// Mempool input channels
			case b := <-m.accepted.blockChan:
				m.onAcceptedBlock(b)
			case tx := <-m.pending:
				m.onPendingTx(tx)
			case <-time.After(20 * time.Second):
				m.onIdle()
			// Mempool terminating
			case <-m.quitChan:
				return
			}
		}
	}()
}

// onPendingTx ensures all transaction rules are satisfied before adding the tx
// into the verified pool
func (m *Mempool) onPendingTx(t TxDesc) {
	// stats to log
	log.Tracef("pending txs count %d", len(m.pending))

	txID, err := t.tx.CalculateHash()
	if err != nil {
		log.Tracef("calculate tx hash failed: %s", err.Error())
		return
	}

	log := logEntry("tx", toHex(txID[:]))

	if t.tx.Type() == transactions.CoinbaseType {
		// coinbase tx should be built by block generator only
		log.Warnf("coinbase tx not allowed")
		return
	}

	// expect it is not already a verified tx
	if m.verified.Contains(txID) {
		log.Warnf("already exists")
		return
	}

	// expect it is not already spent from mempool verified txs
	if err := m.checkTXDoubleSpent(t.tx); err != nil {
		log.Warnf("double-spending: %v", err)
		return
	}

	// execute tx verification procedure
	if err := m.checkTx(t.tx); err != nil {
		log.Errorf("verification: %v", err)
		return
	}

	// if consumer's verification passes, mark it as verified
	t.verified = time.Now()

	// we've got a valid transaction pushed
	if err := m.verified.Put(t); err != nil {
		log.Errorf("store: %v", err)
		return
	}

	// advertise the hash of the verified Tx to the P2P network
	if err := m.advertiseTx(txID); err != nil {
		log.Errorf("advertise: %v", err)
		return
	}
}

func (m *Mempool) onAcceptedBlock(b block.Block) {
	m.latestBlockTimestamp = b.Header.Timestamp
	m.removeAccepted(b)
}

// removeAccepted to clean up all txs from the mempool that have been already
// added to the chain.
//
// Instead of doing a full DB scan, here we rely on the latest accepted block to
// update.
//
// The passed block is supposed to be the last one accepted. That said, it must
// contain a valid TxRoot.
func (m *Mempool) removeAccepted(b block.Block) {

	log.Infof("processing an accepted block with %d txs", len(b.Txs))

	if m.verified.Len() == 0 {
		// No txs accepted then no cleanup needed
		return
	}

	payloads := make([]merkletree.Payload, len(b.Txs))
	for i, tx := range b.Txs {
		payloads[i] = tx.(merkletree.Payload)
	}

	tree, err := merkletree.NewTree(payloads)

	if err == nil && tree != nil {

		if !bytes.Equal(tree.MerkleRoot, b.Header.TxRoot) {
			log.Error("the accepted block seems to have invalid txroot")
			return
		}

		s := m.newPool()
		// Check if mempool verified tx is part of merkle tree of this block
		// if not, then keep it in the mempool for the next block
		err = m.verified.Range(func(k key, t TxDesc) error {
			if r, _ := tree.VerifyContent(t.tx); !r {
				if err := s.Put(t); err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			log.Error(err.Error())
		}

		m.verified = s
	}

	log.Infof("processing completed")
}

func (m *Mempool) onIdle() {

	// stats to log
	log.Infof("verified %d transactions, overall size %.5f MB", m.verified.Len(), m.verified.Size())

	// trigger alarms/notifications in case of abnormal state

	// trigger alarms on too much txs memory allocated
	if m.verified.Size() > float64(config.Get().Mempool.MaxSizeMB) {
		log.Errorf("mempool is full")
	}

	if log.Logger.Level == logger.TraceLevel {
		// print all txs
		var counter int
		log.Tracef("list of the verified txs")
		_ = m.verified.Range(func(k key, t TxDesc) error {
			log.Tracef("tx: %s", toHex(k[:]))
			counter++
			return nil
		})
	}

	// TODO: Get rid of stuck/expired transactions

	// TODO: Check periodically the oldest txs if somehow were accepted into the
	// blockchain but were not removed from mempool verified list.
	/*()
	err = c.db.View(func(t database.Transaction) error {
		_, _, _, err := t.FetchBlockTxByHash(txID)
		return err
	})
	*/
}

func (m *Mempool) newPool() Pool {

	preallocTxs := config.Get().Mempool.PreallocTxs

	var p Pool
	switch config.Get().Mempool.PoolType {
	case "hashmap":
		p = &HashMap{Capacity: preallocTxs}
	case "syncpool":
		panic("syncpool not supported")
	default:
		p = &HashMap{Capacity: preallocTxs}
	}

	return p
}

// Collect process the emitted transactions.
// Fast-processing and simple impl to avoid locking here.
// NB This is always run in a different than main mempool routine
func (m *Mempool) Collect(message *bytes.Buffer) error {

	txs, err := transactions.FromReader(message, 1)
	if err != nil {
		return err
	}

	m.pending <- TxDesc{tx: txs[0], received: time.Now()}

	return nil
}

// onGetMempoolTxs retrieves current state of the mempool of the verified but
// still unaccepted txs
func (m Mempool) onGetMempoolTxs(r wire.Req) {

	// Read inputs
	filterTxID := r.Params.Bytes()

	outputTxs := make([]transactions.Transaction, 0)

	// TODO: When filterTxID is empty, mempool returns the all verified
	// txs. Once the limit of transactions space in a block is determined,
	// mempool should prioritize transactions by fee
	err := m.verified.Range(func(k key, t TxDesc) error {
		if len(filterTxID) > 0 {
			if bytes.Equal(filterTxID, k[:]) {
				// tx found
				outputTxs = append(outputTxs, t.tx)
				return nil
			}
		} else if len(outputTxs) < 50 {
			// Non-filter scan for max 50 transactions.
			// TODO: this should be properly determined ASAP (maybe by adding size checks that determine
			// the amount of kB a transaction takes up)
			outputTxs = append(outputTxs, t.tx)
		}

		return nil
	})

	if err != nil {
		r.ErrChan <- err
		return
	}

	// marshal Txs
	w := new(bytes.Buffer)
	lTxs := uint64(len(outputTxs))
	if err := encoding.WriteVarInt(w, lTxs); err != nil {
		r.ErrChan <- err
		return
	}

	for _, tx := range outputTxs {
		if err := tx.Encode(w); err != nil {
			r.ErrChan <- err
			return
		}
	}

	r.RespChan <- *w
}

// checkTXDoubleSpent differs from verifiers.checkTXDoubleSpent as it executes
// all checks against mempool verified txs but not blockchain db.
func (m *Mempool) checkTXDoubleSpent(tx transactions.Transaction) error {

	for _, input := range tx.StandardTX().Inputs {
		exists := m.verified.ContainsKeyImage(input.KeyImage)
		if exists {
			return errors.New("tx already spent")
		}
	}

	return nil
}

// Quit makes mempool main loop to terminate
func (m *Mempool) Quit() {
	m.quitChan <- struct{}{}
}

// Send Inventory message to all peers
func (m *Mempool) advertiseTx(txID []byte) error {

	msg := &peermsg.Inv{}
	msg.AddItem(peermsg.InvTypeMempoolTx, txID)

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		panic(err)
	}

	withTopic, err := wire.AddTopic(buf, topics.Inv)
	if err != nil {
		return err
	}

	m.eventBus.Stream(string(topics.Gossip), withTopic)
	return nil
}

func toHex(id []byte) string {
	enc := hex.EncodeToString(id[:])
	if len(enc) >= 16 {
		return enc[0:16]
	}
	return enc
}

func logEntry(key, val string) *logger.Entry {
	fields := logger.Fields{}
	// copy default fields
	for key, value := range log.Data {
		fields[key] = value
	}

	fields[key] = val
	return logger.WithFields(fields)
}
