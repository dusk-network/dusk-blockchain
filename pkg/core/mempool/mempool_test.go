package mempool

import (
	"bytes"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// verifier func mock
var verifyFunc = func(tx transactions.Transaction) error {

	// some dummy check to distinguish between valid and non-valid txs for
	// this test
	val := float64(tx.StandardTX().Version)
	if math.Mod(val, 2) != 0 {
		return errors.New("invalid tx version")
	}
	return nil
}

// Collect implements wire.EventCollector.
func (c *ctx) Collect(message *bytes.Buffer) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	msg := *message
	var topicBytes [15]byte

	reader := bytes.NewReader(msg.Bytes())
	_, _ = reader.Read(topicBytes[:])
	topic := topics.ByteArrayToTopic(topicBytes)

	if topic == topics.Tx {
		tx, _ := transactions.FromReader(reader, 1)
		for i := range tx {
			c.propagated = append(c.propagated, tx[i])
		}
	}

	return nil
}

// Helper struct around mempool asserts to shorten common code
var c *ctx

type ctx struct {
	verifiedTx []transactions.Transaction
	propagated []transactions.Transaction
	mu         sync.Mutex
	m          *Mempool

	bus    *wire.EventBus
	rpcBus *wire.RPCBus
}

func initCtx(t *testing.T) *ctx {

	time.Sleep(1 * time.Second)
	if c == nil {
		c = &ctx{}
		c.verifiedTx = make([]transactions.Transaction, 0)

		// config
		r := config.Registry{}
		r.Mempool.MaxSizeMB = 1
		r.Mempool.PoolType = "hashmap"
		config.Mock(&r)
		// eventBus
		c.bus = wire.NewEventBus()
		// creating the rpcbus
		c.rpcBus = wire.NewRPCBus()

		c.propagated = make([]transactions.Transaction, 0)
		go wire.NewTopicListener(c.bus, c, string(topics.Gossip)).Accept()

		// mempool
		c.m = NewMempool(c.bus, verifyFunc)
	} else {

		// Reset shared context state
		c.m.Quit()
		c.m.verified = c.m.newPool()
		c.verifiedTx = make([]transactions.Transaction, 0)
		c.propagated = make([]transactions.Transaction, 0)
	}

	c.m.Run()

	return c
}

func (c *ctx) addTx(tx transactions.Transaction) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := verifyFunc(tx); err == nil {
		c.verifiedTx = append(c.verifiedTx, tx)
	}
}

// Wait until the EventBus chan is drained and all pending txs are fully
// processed. This is important to synchronously compare the expected with the
// yielded results.
func (c *ctx) wait() {
	time.Sleep(500 * time.Millisecond)
}

func (c *ctx) assert(t *testing.T, propagated bool) {

	c.wait()

	r, _ := c.rpcBus.Call(wire.GetVerifiedTxs, wire.NewRequest(bytes.Buffer{}, 3))

	lTxs, _ := encoding.ReadVarInt(&r)
	txs, _ := transactions.FromReader(&r, lTxs)

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(txs) != len(c.verifiedTx) {
		t.Fatalf("expecting %d verified txs but mempool stores %d txs", len(c.verifiedTx), len(txs))
	}

	for i, tx := range c.verifiedTx {

		var exists bool
		for _, memTx := range txs {
			if memTx.Equals(tx) {
				exists = true
				break
			}
		}

		if !exists {
			t.Fatalf("a verified tx not found (index %d)", i)
		}
	}

	if propagated {
		if len(txs) != len(c.propagated) {
			t.Fatalf("expecting %d propagated txs but mempool stores %d txs", len(c.propagated), len(txs))
		}
	}

}

// Only difference with helper.RandomSliceOfTxs is lack of appending a coinbase tx
func randomSliceOfTxs(t *testing.T, txsBatchCount uint16) []transactions.Transaction {
	var txs []transactions.Transaction

	var i uint16
	for ; i < txsBatchCount; i++ {

		txs = append(txs, helper.RandomStandardTx(t, false))
		txs = append(txs, helper.RandomTLockTx(t, false))

		stake, err := helper.RandomStakeTx(t, false)
		assert.Nil(t, err)
		txs = append(txs, stake)

		bid, err := helper.RandomBidTx(t, false)
		assert.Nil(t, err)
		txs = append(txs, bid)
	}

	return txs
}

func TestProcessPendingTxs(t *testing.T) {

	initCtx(t)

	var version uint8
	txs := randomSliceOfTxs(t, 5)

	for _, tx := range txs {

		// Publish valid tx
		buf := new(bytes.Buffer)
		err := tx.Encode(buf)
		if err != nil {
			t.Fatal(err)
		}

		c.addTx(tx)
		c.bus.Publish(string(topics.Tx), buf)

		// Publish invalid txs (one that does not pass verifyTx)
		version++
		tx := transactions.NewStandard(version, 2)
		buf = new(bytes.Buffer)
		err = tx.Encode(buf)
		if err != nil {
			t.Fatal(err)
		}

		c.addTx(tx)
		c.bus.Publish(string(topics.Tx), buf)

		// Publish a duplicated tx
		buf = new(bytes.Buffer)
		_ = tx.Encode(buf)
		c.bus.Publish(string(topics.Tx), buf)
	}

	c.assert(t, true)

}

func TestProcessPendingTxsAsync(t *testing.T) {

	initCtx(t)

	wg := sync.WaitGroup{}
	var version uint64
	for i := 0; i < 3; i++ {

		wg.Add(1)
		// Publish valid txs
		go func() {

			txs := randomSliceOfTxs(t, 3)
			for _, tx := range txs {
				buf := new(bytes.Buffer)
				_ = tx.Encode(buf)

				c.addTx(tx)
				c.bus.Publish(string(topics.Tx), buf)
			}
			wg.Done()
		}()

		wg.Add(1)
		// Publish invalid txs
		go func() {
			for y := 0; y <= 5; y++ {
				buf := new(bytes.Buffer)

				// change version to get valid/invalid txs
				v := atomic.AddUint64(&version, 1)
				tx := transactions.NewStandard(uint8(v), 2)
				_ = tx.Encode(buf)

				c.addTx(tx)
				c.bus.Publish(string(topics.Tx), buf)
			}

			wg.Done()
		}()
	}

	wg.Wait()

	c.assert(t, true)
}

func TestRemoveAccepted(t *testing.T) {

	initCtx(t)

	// Create a random block
	b := helper.RandomBlock(t, 200, 0)
	b.Txs = make([]transactions.Transaction, 0)

	counter := 0

	// generate 3*4 random txs
	txs := randomSliceOfTxs(t, 3)

	for _, tx := range txs {
		buf := new(bytes.Buffer)
		err := tx.Encode(buf)
		if err != nil {
			t.Fatal(err)
		}

		// Publish valid tx
		c.bus.Publish(string(topics.Tx), buf)

		// Simulate a situation where the block has accepted each 2th tx
		counter++
		if math.Mod(float64(counter), 2) == 0 {
			b.AddTx(tx)
			// If tx is accepted, it is expected to be removed from mempool on
			// onAcceptBlock event
		} else {
			c.addTx(tx)
		}
	}

	c.wait()

	_ = b.SetRoot()
	buf := new(bytes.Buffer)
	_ = b.Encode(buf)

	c.bus.Publish(string(topics.AcceptedBlock), buf)

	c.assert(t, false)
}

func TestCoibaseTxsNotAllowed(t *testing.T) {

	initCtx(t)

	// Publish a set of valid txs
	txs := randomSliceOfTxs(t, 1)

	for _, tx := range txs {
		buf := new(bytes.Buffer)
		err := tx.Encode(buf)
		if err != nil {
			t.Fatal(err)
		}

		c.addTx(tx)
		c.bus.Publish(string(topics.Tx), buf)
	}

	// Publish a coinbase txs
	tx := helper.RandomCoinBaseTx(t, false)
	buf := new(bytes.Buffer)
	err := tx.Encode(buf)
	if err != nil {
		t.Fatal(err)
	}

	c.bus.Publish(string(topics.Tx), buf)

	c.wait()

	// Assert that all non-coinbase txs have been verified
	c.assert(t, true)
}
