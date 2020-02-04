package mempool

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/core/verifiers"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/peermsg"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-crypto/merkletree"
	"github.com/dusk-network/dusk-wallet/v2/block"
	"github.com/dusk-network/dusk-wallet/v2/transactions"
	logger "github.com/sirupsen/logrus"
)

var log = logger.WithFields(logger.Fields{"prefix": "mempool"})

const (
	consensusSeconds = 20
	maxPendingLen    = 1000
)

var (
	// ErrCoinbaseTxNotAllowed coinbase tx must be built by block generator only
	ErrCoinbaseTxNotAllowed = errors.New("coinbase tx not allowed")
	// ErrAlreadyExists transaction with same txid already exists in
	ErrAlreadyExists = errors.New("already exists")
	// ErrDoubleSpending transaction uses outputs spent in other mempool txs
	ErrDoubleSpending = errors.New("double-spending in mempool")
)

// Mempool is a storage for the chain transactions that are valid according to the
// current chain state and can be included in the next block.
type Mempool struct {
	getMempoolTxsChan       <-chan rpcbus.Request
	getMempoolTxsBySizeChan <-chan rpcbus.Request
	getMempoolViewChan      <-chan rpcbus.Request
	sendTxChan              <-chan rpcbus.Request

	// transactions emitted by RPC and Peer subsystems
	// pending to be verified before adding them to verified pool
	pending chan TxDesc

	// verified txs to be included in next block
	verified Pool

	// the collector to listen for new intermediate blocks
	intermediateBlockChan <-chan block.Block
	acceptedBlockChan     <-chan block.Block

	// used by tx verification procedure
	latestBlockTimestamp int64

	eventBus *eventbus.EventBus
	db       database.DB

	// the magic function that knows best what is valid chain Tx
	verifyTx func(tx transactions.Transaction) error
	quitChan chan struct{}

	// ID of subscription to the TX topic on the EventBus
	txSubscriberID uint32
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

// NewMempool instantiates and initializes node mempool
func NewMempool(eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus, verifyTx func(tx transactions.Transaction) error) *Mempool {

	log.Infof("Create instance")

	getMempoolTxsChan := make(chan rpcbus.Request, 1)
	if err := rpcBus.Register(rpcbus.GetMempoolTxs, getMempoolTxsChan); err != nil {
		log.Errorf("rpcbus.GetMempoolTxs err=%v", err)
	}

	getMempoolTxsBySizeChan := make(chan rpcbus.Request, 1)
	if err := rpcBus.Register(rpcbus.GetMempoolTxsBySize, getMempoolTxsBySizeChan); err != nil {
		log.Errorf("rpcbus.getMempoolTxsBySize err=%v", err)
	}

	getMempoolViewChan := make(chan rpcbus.Request, 1)
	if err := rpcBus.Register(rpcbus.GetMempoolView, getMempoolViewChan); err != nil {
		log.WithError(err).Errorf("error registering getMempoolView")
	}

	sendTxChan := make(chan rpcbus.Request, 1)
	if err := rpcBus.Register(rpcbus.SendMempoolTx, sendTxChan); err != nil {
		log.Errorf("rpcbus.SendMempoolTx err=%v", err)
	}

	intermediateBlockChan := initIntermediateBlockCollector(eventBus)
	acceptedBlockChan, _ := consensus.InitAcceptedBlockUpdate(eventBus)

	m := &Mempool{
		eventBus:                eventBus,
		latestBlockTimestamp:    math.MinInt32,
		quitChan:                make(chan struct{}),
		intermediateBlockChan:   intermediateBlockChan,
		acceptedBlockChan:       acceptedBlockChan,
		getMempoolTxsChan:       getMempoolTxsChan,
		getMempoolTxsBySizeChan: getMempoolTxsBySizeChan,
		getMempoolViewChan:      getMempoolViewChan,
		sendTxChan:              sendTxChan,
	}

	if verifyTx != nil {
		m.verifyTx = verifyTx
	}

	m.verified = m.newPool()

	log.Infof("Running with pool type %s", config.Get().Mempool.PoolType)

	// topics.Tx will be published by RPC subsystem or Peer subsystem (deserialized from gossip msg)
	m.pending = make(chan TxDesc, maxPendingLen)
	l := eventbus.NewCallbackListener(m.CollectPending)
	m.txSubscriberID = m.eventBus.Subscribe(topics.Tx, l)
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
			//rpcbus methods
			case r := <-m.sendTxChan:
				handleRequest(r, m.onSendMempoolTx, "SendTx")
			case r := <-m.getMempoolTxsChan:
				handleRequest(r, m.onGetMempoolTxs, "GetMempoolTxs")
			case r := <-m.getMempoolTxsBySizeChan:
				handleRequest(r, m.onGetMempoolTxsBySize, "GetMempoolTxsBySize")
			case r := <-m.getMempoolViewChan:
				handleRequest(r, m.onGetMempoolView, "GetMempoolView")
			// Mempool input channels
			case b := <-m.intermediateBlockChan:
				m.onBlock(b)
			case b := <-m.acceptedBlockChan:
				m.onBlock(b)
			case tx := <-m.pending:
				// TODO: the m.pending channel looks a bit wasteful. Consider
				// removing it and call onPendingTx directly within
				// CollectPending
				_, _ = m.onPendingTx(tx)
			case <-time.After(20 * time.Second):
				m.onIdle()
			// Mempool terminating
			case <-m.quitChan:
				//m.eventBus.Unsubscribe(topics.Tx, m.txSubscriberID)
				return
			}
		}
	}()
}

// onPendingTx handles a submitted tx from any source (rpcBus or eventBus)
func (m *Mempool) onPendingTx(t TxDesc) ([]byte, error) {

	log.Infof("Pending txs=%d", len(m.pending))

	start := time.Now()
	txid, err := m.processTx(t)
	elapsed := time.Since(start)

	if err != nil {
		log.Errorf("Failed txid=%s err='%v' duration=%d μs", toHex(txid), err, elapsed.Microseconds())
	} else {
		log.Infof("Verified txid=%s duration=%d μs", toHex(txid), elapsed.Microseconds())
	}

	return txid, err
}

// processTx ensures all transaction rules are satisfied before adding the tx
// into the verified pool
func (m *Mempool) processTx(t TxDesc) ([]byte, error) {

	txid, err := t.tx.CalculateHash()
	if err != nil {
		return txid, fmt.Errorf("hash err: %s", err.Error())
	}

	log.Infof("Pending txid=%s size=%d bytes", toHex(txid), t.size)

	if t.tx.Type() == transactions.CoinbaseType {
		// coinbase tx should be built by block generator only
		return txid, ErrCoinbaseTxNotAllowed
	}

	// expect it is not already a verified tx
	if m.verified.Contains(txid) {
		return txid, ErrAlreadyExists
	}

	// expect it is not already spent from mempool verified txs
	if err := m.checkTXDoubleSpent(t.tx); err != nil {
		return txid, ErrDoubleSpending
	}

	// execute tx verification procedure
	if err := m.checkTx(t.tx); err != nil {
		return txid, fmt.Errorf("verification: %v", err)
	}

	// if consumer's verification passes, mark it as verified
	t.verified = time.Now()

	// we've got a valid transaction pushed
	if err := m.verified.Put(t); err != nil {
		return txid, fmt.Errorf("store: %v", err)
	}

	// advertise the hash of the verified tx to the P2P network
	if err := m.advertiseTx(txid); err != nil {
		// TODO: Perform re-advertise procedure
		return txid, fmt.Errorf("advertise: %v", err)
	}

	return txid, nil
}

func (m *Mempool) onBlock(b block.Block) {
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

	blockHash := toHex(b.Header.Hash)

	log.Infof("Processing block %s with %d txs", blockHash, len(b.Txs))

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
			log.Errorf("block %s has invalid txroot", blockHash)
			return
		}

		s := m.newPool()
		// Check if mempool verified tx is part of merkle tree of this block
		// if not, then keep it in the mempool for the next block
		err = m.verified.Range(func(k txHash, t TxDesc) error {
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

	log.Infof("Processing block %s completed", toHex(b.Header.Hash))
}

func (m *Mempool) onIdle() {

	// stats to log
	poolSize := float32(m.verified.Size()) / 1000
	log.Infof("Txs count %d, total size %.3f kB", m.verified.Len(), poolSize)

	// trigger alarms/notifications in case of abnormal state

	// trigger alarms on too much txs memory allocated
	maxSizeBytes := config.Get().Mempool.MaxSizeMB * 1000 * 1000
	if m.verified.Size() > maxSizeBytes {
		log.Warnf("Mempool is bigger than %d MB", config.Get().Mempool.MaxSizeMB)
	}

	if log.Logger.Level == logger.TraceLevel {
		if m.verified.Len() > 0 {
			_ = m.verified.Range(func(k txHash, t TxDesc) error {
				log.Tracef("txid=%s", toHex(k[:]))
				return nil
			})
		}
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
		log.Panic("syncpool not supported")
	default:
		p = &HashMap{Capacity: preallocTxs}
	}

	return p
}

// CollectPending process the emitted transactions.
// Fast-processing and simple impl to avoid locking here.
// NB This is always run in a different than main mempool routine
func (m *Mempool) CollectPending(msg message.Message) error {
	tx := msg.Payload().(transactions.Transaction)
	m.pending <- TxDesc{tx: tx, received: time.Now(), size: uint(len(msg.Id()))}
	return nil
}

// onGetMempoolTxs retrieves current state of the mempool of the verified but
// still unaccepted txs.
// Called by P2P on InvTypeMempoolTx msg
func (m Mempool) onGetMempoolTxs(r rpcbus.Request) (bytes.Buffer, error) {
	// Read inputs
	filterTxID := r.Params.Bytes()

	outputTxs := make([]transactions.Transaction, 0)
	w := new(bytes.Buffer)

	// If we are looking for a specific tx, just look it up by key.
	if len(filterTxID) == 32 {
		tx := m.verified.Get(filterTxID)
		if tx == nil {
			return bytes.Buffer{}, errors.New("tx not found")
		}

		outputTxs = append(outputTxs, tx)
		err := marshalTxs(w, outputTxs)
		return *w, err
	}

	// When filterTxID is empty, mempool returns all verified txs sorted
	// by fee from highest to lowest
	err := m.verified.RangeSort(func(k txHash, t TxDesc) (bool, error) {
		outputTxs = append(outputTxs, t.tx)
		return false, nil
	})

	if err != nil {
		return bytes.Buffer{}, err
	}

	err = marshalTxs(w, outputTxs)
	return *w, err
}

func (m Mempool) onGetMempoolView(r rpcbus.Request) (bytes.Buffer, error) {
	txs := make([]transactions.Transaction, 0)
	switch len(r.Params.Bytes()) {
	case 32:
		// If we want a tx with a certain ID, we can simply look it up
		// directly
		hash, err := hex.DecodeString(string(r.Params.Bytes()))
		if err != nil {
			return bytes.Buffer{}, err
		}

		tx := m.verified.Get(hash)
		if tx == nil {
			return bytes.Buffer{}, errors.New("tx not found")
		}

		txs = append(txs, tx)
	case 1:
		txs = m.verified.FilterByType(transactions.TxType(r.Params.Bytes()[0]))
	default:
		txs = m.verified.Clone()
	}

	buf := new(bytes.Buffer)
	for _, tx := range txs {
		if _, err := buf.WriteString(fmt.Sprintf("Type: %v / Hash: %s / Locktime: %v\n", tx.Type(), hex.EncodeToString(tx.StandardTx().TxID), tx.LockTime())); err != nil {
			return bytes.Buffer{}, err
		}
	}

	return *buf, nil
}

// onGetMempoolTxsBySize returns a subset of verified mempool txs which
// 1. contains only highest fee txs
// 2. has total txs size not bigger than maxTxsSize (request param)
// Called by BlockGenerator on generating a new candidate block
func (m Mempool) onGetMempoolTxsBySize(r rpcbus.Request) (bytes.Buffer, error) {

	// Read maxTxsSize param
	var maxTxsSize uint32
	if err := encoding.ReadUint32LE(&r.Params, &maxTxsSize); err != nil {
		return bytes.Buffer{}, err
	}

	txs := make([]transactions.Transaction, 0)

	var totalSize uint32
	err := m.verified.RangeSort(func(k txHash, t TxDesc) (bool, error) {

		var done bool
		totalSize += uint32(t.size)

		if totalSize <= maxTxsSize {
			txs = append(txs, t.tx)
		} else {
			done = true
		}

		return done, nil
	})

	if err != nil {
		return bytes.Buffer{}, err
	}

	w := new(bytes.Buffer)
	err = marshalTxs(w, txs)
	return *w, err
}

// onSendMempoolTx utilizes rpcbus to allow submitting a tx to mempool with
func (m Mempool) onSendMempoolTx(r rpcbus.Request) (bytes.Buffer, error) {

	txDesc, err := unmarshalTxDesc(r.Params)
	if err != nil {
		return bytes.Buffer{}, err
	}

	// Process request
	txid, err := m.onPendingTx(txDesc)

	result := bytes.Buffer{}
	result.Write(txid)

	return result, err
}

// checkTXDoubleSpent differs from verifiers.checkTXDoubleSpent as it executes on
// all checks against mempool verified txs but not blockchain db.on
func (m *Mempool) checkTXDoubleSpent(tx transactions.Transaction) error {
	for _, input := range tx.StandardTx().Inputs {

		exists := m.verified.ContainsKeyImage(input.KeyImage.Bytes())
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

	// TODO: can we simply encode the message directly on a topic carrying buffer?
	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		log.Panic(err)
	}

	if err := topics.Prepend(buf, topics.Inv); err != nil {
		log.Panic(err)
	}

	// TODO: interface - marshalling should done after the Gossip, not before
	packet := message.New(topics.Inv, *buf)
	m.eventBus.Publish(topics.Gossip, packet)
	return nil
}

func toHex(id []byte) string {
	enc := hex.EncodeToString(id[:])
	return enc
}

func unmarshalTxDesc(m bytes.Buffer) (TxDesc, error) {

	bufSize := m.Len()
	tx, err := message.UnmarshalTx(&m)
	if err != nil {
		return TxDesc{}, err
	}

	// txSize is number of unmarshaled bytes
	txSize := bufSize - m.Len()

	return TxDesc{tx: tx, received: time.Now(), size: uint(txSize)}, nil
}

func handleRequest(r rpcbus.Request, handler func(r rpcbus.Request) (bytes.Buffer, error), name string) {

	log.Tracef("Handling %s request", name)

	result, err := handler(r)
	if err != nil {
		log.Errorf("Failed %s request: %v", name, err)
		r.RespChan <- rpcbus.Response{Err: err}
		return
	}

	r.RespChan <- rpcbus.Response{Resp: result, Err: nil}

	log.Tracef("Handled %s request", name)
}

func marshalTxs(w *bytes.Buffer, txs []transactions.Transaction) error {
	lTxs := uint64(len(txs))
	if err := encoding.WriteVarInt(w, lTxs); err != nil {
		return err
	}

	for _, tx := range txs {
		if err := message.MarshalTx(w, tx); err != nil {
			return err
		}
	}

	return nil
}
