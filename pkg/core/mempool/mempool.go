package mempool

import (
	"bytes"
	logger "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"sync"
	"time"
)

var log *logger.Entry = logger.WithFields(logger.Fields{"prefix": "mempool"})

// Mempool is a storage for the chain transactions that are valid according to the
// current chain state and can be included in the next block.
type Mempool struct {

	// transactions emitted by RPC and Peer subsystems
	// pending to be verified before adding them to verified pool
	pending chan TxDesc

	// verified txs to be included in next block
	verified Pool
	// mutex to guard the verified pool
	mu sync.RWMutex

	// the magic function that knows best what is a valid transaction
	verifyTx func(transactions.Transaction) error

	eventBus *wire.EventBus
}

func NewMempool(eventBus *wire.EventBus, verifyTx func(transactions.Transaction) error) *Mempool {

	log.Infof("Create new instance")

	m := &Mempool{verifyTx: verifyTx, eventBus: eventBus}

	if m.verifyTx == nil {
		panic("invalid verifier function")
	}

	m.verified = m.newPool()

	// topics.Tx will be emitted by RPC subsystem or Peer subsystem (deserialized from gossip msg)
	m.pending = make(chan TxDesc, 100)
	go wire.NewEventSubscriber(eventBus, m, string(topics.Tx)).Accept()

	// Spawn mempool lifecycle routine
	go func() {
		for {
			select {
			case tx := <-m.pending:
				m.onPendingTx(tx)
			case <-time.After(3 * time.Second):
				m.onIdle()
			}
		}
	}()

	log.Infof("Running with pool type %s", config.Get().Mempool.PoolType)

	return m
}

// GetVerifiedTxs to be called by any subsystem that requires the latest
// snapshot of verified txs. It's concurrent-safe. Block proposer and RPC
// service could be the consumers for it.
func (m *Mempool) GetVerifiedTxs() []transactions.Transaction {

	// TODO: Two more options here to spread the good news we have verified txs
	//
	// 1. EventBus topic to be published but that might cause inconsistency and
	// many times duplicated mempool
	//
	// 2. Make it possible for any subsystem to read verifiedTxs from the blockchain DB

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.verified.Clone()
}

func (m *Mempool) onPendingTx(t TxDesc) {

	// stats to log
	log.Tracef("Stats: pending %d", len(m.pending))

	m.mu.Lock()
	defer m.mu.Unlock()

	txID, _ := t.tx.CalculateHash()

	if t.tx.Type() == transactions.CoinbaseType {
		// coinbase tx should be built by block generator only
		log.Warnf("Coinbase tx not allowed")
		return
	}

	// expect it is not already accepted tx
	if m.verified.Contains(txID) {
		log.Warnf("Duplicated tx")
		return
	}

	// delegate the verification procedure to the consumer
	err := m.verifyTx(t.tx)

	if err != nil {
		log.Errorf("Tx verification error: %v", err)
		return
	}

	// if consumer's verification passes, mark it as verified
	t.verified = time.Now()

	// we've got a valid transaction pushed
	if err := m.verified.Put(t); err != nil {
		log.Error(err.Error())
	}

	// propagate to P2P network
	buffer := new(bytes.Buffer)

	topicBytes := topics.TopicToByteArray(topics.Tx)
	if _, err := buffer.Write(topicBytes[:]); err != nil {
		log.Errorf("%v", err)
		return
	}

	if err := t.tx.Encode(buffer); err != nil {
		log.Errorf("%v", err)
		return
	}

	m.eventBus.Publish(string(topics.Gossip), buffer)
}

// RemoveAccepted to clean up all txs from the mempool that have been already
// added to the chain.
//
// Instead of doing a full DB scan, here we rely on the latest accepted block to
// update.
//
// The passed block is supposed to be the last one accepted. That said, it must
// contain a valid TxRoot.
func (m *Mempool) RemoveAccepted(b block.Block) {

	m.mu.Lock()
	defer m.mu.Unlock()

	log.Infof("New verified block with %d txs being processed", len(b.Txs))

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
			log.Error("The accepted block seems to have invalid TxRoot")
			return
		}

		s := m.newPool()
		// Check if mempool verified tx is part of merkle tree of this block
		// if not, then keep it in the mempool for the next block
		err = m.verified.Range(func(k key, t TxDesc) error {
			if r, _ := tree.VerifyContent(t.tx); r {
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

	// TODO: Save the timing data per tx - received, verified, accepted if all
	// available
}

func (m *Mempool) onIdle() {

	// stats to log
	log.Infof("Stats: verified %d, mem %d MB", m.verified.Len(), m.verified.Size())

	// Cleanup stuck transactions
	// TODO:

	// trigger alarms/notifications in casae of abnormal state

	// trigger alarms on too much txs memory allocated
	if m.verified.Size() > config.Get().Mempool.MaxSizeMB {
		log.Errorf("Mempool is full")
	}

	// Measure average time difference between recieved and verified
	// TODO:

	// Check oldest txs if somehow were accepted into the blockchain but were
	// not removed from mempool verified. This should be

	/*()
	err = c.db.View(func(t database.Transaction) error {
		_, _, _, err := t.FetchBlockTxByHash(txID)
		return err
	})
	*/

}

func (m *Mempool) newPool() Pool {

	// TODO: Measure average block txs to determine best preallocTxs dynamically
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
