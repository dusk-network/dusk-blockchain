package mempool

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"

	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/transactions"
	"github.com/stretchr/testify/assert"
)

// verifier func mock
var verifyFunc = func(tx transactions.Transaction) error {

	// some dummy check to distinguish between valid and non-valid txs for
	// this test
	val := float64(tx.StandardTx().Version)
	if math.Mod(val, 2) != 0 {
		return errors.New("invalid tx version")
	}
	return nil
}

// Helper struct around mempool asserts to shorten common code
var c *ctx

// ctx main role is to collect the expected verified and propagated txs so that
// it can assert that mempool has the proper set of txs after particular events
type ctx struct {
	verifiedTx []transactions.Transaction
	propagated [][]byte
	mu         sync.Mutex

	m      *Mempool
	bus    *eventbus.EventBus
	rpcBus *rpcbus.RPCBus
}

func (c *ctx) reset() {

	// Reset shared context state
	c.m.Quit()
	c.m.verified = c.m.newPool()
	c.verifiedTx = make([]transactions.Transaction, 0)
	c.propagated = make([][]byte, 0)

	c.m.Run()
}

func TestMain(m *testing.M) {

	c = &ctx{}

	// config
	r := config.Registry{}
	r.Mempool.MaxSizeMB = 1
	r.Mempool.PoolType = "hashmap"
	r.Mempool.MaxInvItems = 10000
	config.Mock(&r)

	var streamer *eventbus.GossipStreamer
	c.bus, streamer = eventbus.CreateGossipStreamer()
	c.rpcBus = rpcbus.New()

	go func(streamer *eventbus.GossipStreamer, c *ctx) {
		for {
			tx, err := streamer.Read()
			if err != nil {
				fmt.Errorf(err.Error())
			}

			c.mu.Lock()
			c.propagated = append(c.propagated, tx)
			c.mu.Unlock()
		}
	}(streamer, c)

	// initiate a mempool with custom verification function
	c.m = NewMempool(c.bus, c.rpcBus, verifyFunc)
	c.m.Run()

	code := m.Run()
	os.Exit(code)
}

// adding tx in ctx means mempool is expected to store it in the verified list
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

func (c *ctx) assert(t *testing.T, checkPropagated bool) {

	c.wait()

	r, _ := c.rpcBus.Call(rpcbus.GetMempoolTxs, rpcbus.NewRequest(bytes.Buffer{}), 1*time.Second)

	lTxs, _ := encoding.ReadVarInt(&r)

	txs := make([]transactions.Transaction, lTxs)
	for i := uint64(0); i < lTxs; i++ {
		tx, err := marshalling.UnmarshalTx(&r)
		if err != nil {
			t.Fatal(err)
		}
		txs[i] = tx
	}

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
			// ctx is expected to have the same list of verified txs as mempool stores
			t.Fatalf("a verified tx not found (index %d)", i)
		}
	}

	if checkPropagated {
		if len(txs) != len(c.propagated) {
			t.Fatalf("expecting %d propagated txs but mempool stores %d txs", len(c.propagated), len(txs))
		}
	}
}

func TestProcessPendingTxs(t *testing.T) {

	c.reset()

	txs := randomSliceOfTxs(t, 5)
	for _, tx := range txs {

		// Publish valid tx
		buf := new(bytes.Buffer)
		err := marshalling.MarshalTx(buf, tx)
		if err != nil {
			t.Fatal(err)
		}

		c.addTx(tx)
		c.bus.Publish(topics.Tx, buf)

		// Publish invalid/valid txs (ones that do not pass verifyTx and ones that do)
		tx := helper.RandomStandardTx(t, false)
		if err != nil {
			t.Fatal(err)
		}
		tx.Version++
		buf = new(bytes.Buffer)
		err = marshalling.MarshalTx(buf, tx)
		if err != nil {
			t.Fatal(err)
		}

		c.addTx(tx)
		c.bus.Publish(topics.Tx, buf)

		// Publish a duplicated tx
		buf = new(bytes.Buffer)
		_ = marshalling.MarshalTx(buf, tx)
		c.addTx(tx)
		c.bus.Publish(topics.Tx, buf)
	}

	c.assert(t, true)

}

func TestProcessPendingTxsAsync(t *testing.T) {

	c.reset()

	// A batch consists of all 4 types of Dusk transactions (excluding coinbase)
	// The number of batches is the number of concurrent routines that will be
	// publishing the batch txs one by one. To avoid race conditions, as first
	// step we store here all txs that are expected to be in mempool later after
	// go-routines publishing
	batchCount := 8
	const numOfTxsPerBatch = 4
	// generate and store txs that are expected to be valid
	for i := 0; i <= batchCount; i++ {

		// Generate a single batch of txs and added to the expected list of verified
		txs := randomSliceOfTxs(t, 1)
		for _, tx := range txs {
			c.addTx(tx)
		}
	}

	wg := sync.WaitGroup{}

	// Publish valid txs in concurrent manner
	for i := 0; i <= batchCount; i++ {

		// get a slice of all txs
		from := numOfTxsPerBatch * i
		to := from + numOfTxsPerBatch

		wg.Add(1)
		go func(txs []transactions.Transaction) {
			for _, tx := range txs {
				buf := new(bytes.Buffer)
				_ = marshalling.MarshalTx(buf, tx)
				c.bus.Publish(topics.Tx, buf)
			}
			wg.Done()
		}(c.verifiedTx[from:to])
	}

	for i := 0; i < 2; i++ {
		wg.Add(1)
		// Publish invalid txs
		go func() {
			for y := 0; y <= 5; y++ {
				buf := new(bytes.Buffer)
				tx := helper.RandomStandardTx(t, false)
				tx.Version++
				_ = marshalling.MarshalTx(buf, tx)

				c.bus.Publish(topics.Tx, buf)
			}
			wg.Done()
		}()
	}

	wg.Wait()
	c.wait()

	c.assert(t, true)
}

func TestRemoveAccepted(t *testing.T) {

	c.reset()

	// Create a random block
	b := helper.RandomBlock(t, 200, 0)
	b.Txs = make([]transactions.Transaction, 0)

	counter := 0

	// generate 3*4 random txs
	txs := randomSliceOfTxs(t, 3)

	for _, tx := range txs {
		buf := new(bytes.Buffer)
		err := marshalling.MarshalTx(buf, tx)
		if err != nil {
			t.Fatal(err)
		}

		// Publish valid tx
		c.bus.Publish(topics.Tx, buf)

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
	_ = marshalling.MarshalBlock(buf, b)

	c.bus.Publish(topics.IntermediateBlock, buf)

	c.assert(t, false)
}

// TestDoubleSpent ensures mempool rejects txs with keyImages that have been
// already spent from other transactions in the pool.
func TestDoubleSpent(t *testing.T) {

	c.reset()

	// generate 3*4 random txs
	txs := randomSliceOfTxs(t, 3)

	for _, tx := range txs {
		buf := new(bytes.Buffer)
		err := marshalling.MarshalTx(buf, tx)
		if err != nil {
			t.Fatal(err)
		}

		// Publish valid tx
		c.bus.Publish(topics.Tx, buf)
		c.addTx(tx)
	}

	// Create double-spent tx by replicating an already added txs but with
	// differnt TxID
	tx := helper.RandomStandardTx(t, false)

	// Inputs
	tx.Inputs = txs[0].StandardTx().Inputs

	// Outputs
	tx.Outputs = txs[0].StandardTx().Outputs

	buf := new(bytes.Buffer)
	err := marshalling.MarshalTx(buf, tx)
	if err != nil {
		t.Fatal(err)
	}

	// Publish valid tx
	c.bus.Publish(topics.Tx, buf)
	// c.addTx(tx) do not add it into the expected list

	c.wait()

	c.assert(t, false)
}

func TestCoinbaseTxsNotAllowed(t *testing.T) {

	c.reset()

	// Publish a set of valid txs
	txs := randomSliceOfTxs(t, 1)

	for _, tx := range txs {
		buf := new(bytes.Buffer)
		err := marshalling.MarshalTx(buf, tx)
		if err != nil {
			t.Fatal(err)
		}

		c.addTx(tx)
		c.bus.Publish(topics.Tx, buf)
	}

	// Publish a coinbase txs
	tx := helper.RandomCoinBaseTx(t, false)
	buf := new(bytes.Buffer)
	err := marshalling.MarshalTx(buf, tx)
	if err != nil {
		t.Fatal(err)
	}

	c.bus.Publish(topics.Tx, buf)

	c.wait()

	// Assert that all non-coinbase txs have been verified
	c.assert(t, true)
}

func TestSendMempoolTx(t *testing.T) {

	c.reset()

	txs := randomSliceOfTxs(t, 4)

	var totalSize uint32
	for _, tx := range txs {
		buf := new(bytes.Buffer)
		err := marshalling.MarshalTx(buf, tx)
		if err != nil {
			t.Fatal(err)
		}

		totalSize += uint32(buf.Len())

		txidBytes, err := c.rpcBus.Call(rpcbus.SendMempoolTx, rpcbus.NewRequest(*buf), 0)
		if err != nil {
			t.Fatal(err.Error())
		}

		txid, _ := tx.CalculateHash()
		if !bytes.Equal(txidBytes.Bytes(), txid) {
			t.Fatal("unexpected txid retrieved")
		}
	}

	if c.m.verified.Size() != totalSize {
		t.Fatal("unexpected tx total size")
	}

}

func TestMempoolView(t *testing.T) {
	c.reset()

	numTxs := 4
	txs := randomSliceOfTxs(t, uint16(numTxs))

	for _, tx := range txs {
		buf := new(bytes.Buffer)
		err := marshalling.MarshalTx(buf, tx)
		if err != nil {
			t.Fatal(err)
		}

		c.addTx(tx)
		c.bus.Publish(topics.Tx, buf)
	}

	// First, let's just get the entire view
	buf, err := c.rpcBus.Call(rpcbus.GetMempoolView, rpcbus.Request{bytes.Buffer{}, make(chan rpcbus.Response, 1)}, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	// There should be 16 txs, so 16 lines
	n := strings.Count(buf.String(), "\n")
	if n != numTxs*4 {
		t.Fatalf("did not have the required amount of lines in mempool view - %v/%v", n, numTxs*4)
	}

	// Now, we single out one hash from the bunch
	hash := txs[7].StandardTx().TxID
	buf, err = c.rpcBus.Call(rpcbus.GetMempoolView, rpcbus.Request{*bytes.NewBuffer(hash), make(chan rpcbus.Response, 1)}, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	// Should give us info about said tx
	if !strings.Contains(buf.String(), hex.EncodeToString(hash)) {
		t.Fatal("should have gotten info about requested tx")
	}

	// Let's filter for just stakes
	buf, err = c.rpcBus.Call(rpcbus.GetMempoolView, rpcbus.Request{*bytes.NewBuffer([]byte{2}), make(chan rpcbus.Response, 1)}, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	// Should have `numTxs` lines, as there is one tx per type
	// per batch.
	n = strings.Count(buf.String(), "\n")
	if n != numTxs {
		t.Fatalf("did not have required amount of lines when fetching stake txs - %v/%v", n, numTxs)
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
