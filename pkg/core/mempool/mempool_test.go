package mempool

import (
	"bytes"
	"errors"
	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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
type ctx struct {
	verifiedTx []transactions.Transaction
	propagated []transactions.Transaction
	mu         sync.Mutex
	m          *Mempool
	t          *testing.T
	bus        *wire.EventBus
}

func newCtx(t *testing.T) *ctx {
	c := ctx{}
	c.verifiedTx = make([]transactions.Transaction, 0)

	c.t = t
	// config
	r := config.Registry{}
	r.Mempool.MaxSizeMB = 1
	r.Mempool.PoolType = "hashmap"
	config.Mock(&r)
	// eventBus
	c.bus = wire.NewEventBus()

	c.propagated = make([]transactions.Transaction, 0)
	go wire.NewEventSubscriber(c.bus, &c, string(topics.Gossip)).Accept()

	// mempool
	c.m = NewMempool(c.bus, verifyFunc)
	c.assert()
	return &c
}

func (c *ctx) addTx(tx transactions.Transaction) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := verifyFunc(tx); err == nil {
		c.verifiedTx = append(c.verifiedTx, tx)
	}
}

func (c *ctx) assert() {

	// Wait until all pending txs are fully processed
	for {
		if len(c.m.pending) == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// TODO: deep compare
	time.Sleep(50 * time.Millisecond)
	txs := c.m.GetVerifiedTxs()

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(txs) != len(c.verifiedTx) {
		c.t.Fatalf("expecting %d verified txs but mempool stores %d txs", len(c.verifiedTx), len(txs))
	}

	if len(txs) != len(c.propagated) {
		c.t.Fatalf("expecting %d propagated txs but mempool stores %d txs", len(c.propagated), len(txs))
	}

}

func (c *ctx) publish(buf *bytes.Buffer) {

	c.bus.Publish(string(topics.Tx), buf)

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

	c := newCtx(t)

	var version uint8
	for i := 0; i < 2; i++ {

		txs := randomSliceOfTxs(t, 3)
		// Publish valid tx
		for _, tx := range txs {
			buf := new(bytes.Buffer)
			err := tx.Encode(buf)
			if err != nil {
				t.Fatal(err)
			}

			c.addTx(tx)

			c.publish(buf)
		}

		c.assert()

		// Publish invalid tx
		for y := 0; y <= 10; y++ {
			buf := new(bytes.Buffer)

			// set invalid version according to the validator func mock
			version++
			tx := transactions.NewStandard(version, 2)
			err := tx.Encode(buf)
			if err != nil {
				t.Fatal(err)
			}

			c.addTx(tx)

			c.publish(buf)

			// republish same tx to simulate duplicated txs err
			c.publish(buf)
		}

		c.assert()
	}

}

func TestProcessPendingTxsAsync(t *testing.T) {

	c := newCtx(t)

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

				c.publish(buf)
			}
		}()

		wg.Add(1)
		// Publish invalid txs
		go func() {
			for y := 0; y <= 7; y++ {
				buf := new(bytes.Buffer)

				// change version to get valid/invalid txs
				v := atomic.AddUint64(&version, 1)
				tx := transactions.NewStandard(uint8(v), 2)
				_ = tx.Encode(buf)
				c.addTx(tx)

				c.publish(buf)
			}
		}()
	}

	wg.Done()

	time.Sleep(200 * time.Millisecond)
	c.assert()
}

func TestRemoveAccepted(t *testing.T) {

	t.Skip()

	// Create a random block
	b := helper.RandomBlock(t, 200, 2)

	c := newCtx(t)

	counter := 0

	// generate 5*4 random txs
	txs := randomSliceOfTxs(t, 3)

	for _, tx := range txs {
		buf := new(bytes.Buffer)
		err := tx.Encode(buf)
		if err != nil {
			t.Fatal(err)
		}

		// Publish valid tx
		c.publish(buf)

		// Simulate a situation where the block has accepted each 2th tx
		counter++
		if math.Mod(float64(counter), 2) == 0 {
			b.AddTx(tx)
			// If tx is accepted, we expect to be removed from mempool on calling
			// RemoveAccepted()
		} else {
			c.addTx(tx)
		}

	}

	time.Sleep(100 * time.Millisecond)

	_ = b.SetRoot()
	c.m.RemoveAccepted(*b)

	c.assert()

}

// TODO Coinbase ignore
