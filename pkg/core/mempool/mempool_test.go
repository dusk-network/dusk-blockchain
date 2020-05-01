package mempool

import (
	"bytes"
	"context"
	"encoding/hex"
	"math"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/stretchr/testify/assert"
)

// verifier func mock
var verifyFunc = func(tx transactions.ContractCall) error {
	// TODO: come up with something else for RUSK integration

	/*
		// some dummy check to distinguish between valid and non-valid txs for
		// this test
		val := float64(tx.StandardTx().Version)
		if math.Mod(val, 2) != 0 {
			return errors.New("invalid tx version")
		}
		return nil
	*/
	return nil
}

// Helper struct around mempool asserts to shorten common code
var c *ctx

// ctx main role is to collect the expected verified and propagated txs so that
// it can assert that mempool has the proper set of txs after particular events
type ctx struct {
	verifiedTx []transactions.ContractCall
	propagated [][]byte
	mu         sync.Mutex

	m      *Mempool
	bus    *eventbus.EventBus
	rpcBus *rpcbus.RPCBus
}

func (c *ctx) reset() {

	// Reset shared context state
	c.mu.Lock()
	c.m.Quit()
	c.m.verified = c.m.newPool()
	c.verifiedTx = make([]transactions.ContractCall, 0)
	c.propagated = make([][]byte, 0)
	c.mu.Unlock()

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
				log.Panic(err)
			}

			c.mu.Lock()
			c.propagated = append(c.propagated, tx)
			c.mu.Unlock()
		}
	}(streamer, c)

	// initiate a mempool with custom verification function
	c.m = NewMempool(c.bus, c.rpcBus, verifyFunc, nil)
	c.m.Run()

	code := m.Run()
	os.Exit(code)
}

// adding tx in ctx means mempool is expected to store it in the verified list
func (c *ctx) addTx(tx transactions.ContractCall) {
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

	resp, _ := c.rpcBus.Call(topics.GetMempoolTxs, rpcbus.NewRequest(bytes.Buffer{}), 1*time.Second)
	txs := resp.([]transactions.ContractCall)

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(txs) != len(c.verifiedTx) {
		t.Fatalf("expecting %d verified txs but mempool stores %d txs", len(c.verifiedTx), len(txs))
	}

	for i := range c.verifiedTx {

		var exists bool
		// TODO: update for RUSK
		// for _, memTx := range txs {
		// 	if memTx.Equals(tx) {
		// 		exists = true
		// 		break
		// 	}
		// }

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

func prepTx(tx transactions.ContractCall) message.Message {
	return message.New(topics.Tx, tx)
}

func TestProcessPendingTxs(t *testing.T) {

	c.reset()

	txs := randomSliceOfTxs(t, 5)
	for _, tx := range txs {

		// Publish valid tx
		txMsg := prepTx(tx)
		c.addTx(tx)
		c.bus.Publish(topics.Tx, txMsg)

		// Publish invalid/valid txs (ones that do not pass verifyTx and ones that do)
		tx := helper.RandomStandardTx(t, false)
		txMsg = prepTx(tx)
		c.addTx(tx)
		c.bus.Publish(topics.Tx, txMsg)

		// Publish a duplicated tx
		c.addTx(tx)
		c.bus.Publish(topics.Tx, txMsg)
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
		go func(txs []transactions.ContractCall) {
			for _, tx := range txs {
				txMsg := prepTx(tx)
				c.bus.Publish(topics.Tx, txMsg)
			}
			wg.Done()
		}(c.verifiedTx[from:to])
	}

	for i := 0; i < 2; i++ {
		wg.Add(1)
		// Publish invalid txs
		go func() {
			for y := 0; y <= 5; y++ {
				tx := helper.RandomStandardTx(t, false)
				txMsg := prepTx(tx)
				c.bus.Publish(topics.Tx, txMsg)
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
	b.Txs = make([]transactions.ContractCall, 0)

	counter := 0

	// generate 3*4 random txs
	txs := randomSliceOfTxs(t, 3)

	for _, tx := range txs {
		// We avoid sharing this pointer between the mempool and the block
		// by marshaling and unmarshaling the tx
		buf := new(bytes.Buffer)
		if err := message.MarshalTx(buf, tx); err != nil {
			t.Fatal(err)
		}

		txCopy, err := message.UnmarshalTx(buf)
		if err != nil {
			t.Fatal(err)
		}
		txMsg := prepTx(txCopy)

		// Publish valid tx
		c.bus.Publish(topics.Tx, txMsg)

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

	root, _ := b.CalculateRoot()
	b.Header.TxRoot = root
	blockMsg := message.New(topics.IntermediateBlock, *b)
	c.bus.Publish(topics.IntermediateBlock, blockMsg)

	c.assert(t, false)
}

// TestDoubleSpent ensures mempool rejects txs with keyImages that have been
// already spent from other transactions in the pool.
func TestDoubleSpent(t *testing.T) {

	c.reset()

	// generate 3*4 random txs
	txs := randomSliceOfTxs(t, 3)

	for _, tx := range txs {
		txMsg := prepTx(tx)

		// Publish valid tx
		c.bus.Publish(topics.Tx, txMsg)
		c.addTx(tx)
	}

	// Create double-spent tx by replicating an already added txs but with
	// different TxID
	tx := helper.RandomStandardTx(t, false)

	// Inputs
	tx.Inputs = txs[0].StandardTx().Inputs

	// Outputs
	tx.Outputs = txs[0].StandardTx().Outputs
	txMsg := prepTx(tx)

	// Publish valid tx
	c.bus.Publish(topics.Tx, txMsg)
	// c.addTx(tx) do not add it into the expected list

	c.wait()

	c.assert(t, false)
}

func TestCoinbaseTxsNotAllowed(t *testing.T) {

	c.reset()

	// Publish a set of valid txs
	txs := randomSliceOfTxs(t, 1)

	for _, tx := range txs {
		txMsg := prepTx(tx)
		c.addTx(tx)
		c.bus.Publish(topics.Tx, txMsg)
	}

	// Publish a coinbase txs
	tx := helper.RandomCoinBaseTx(t, false)
	txMsg := prepTx(tx)
	c.bus.Publish(topics.Tx, txMsg)

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
		err := message.MarshalTx(buf, tx)
		if err != nil {
			t.Fatal(err)
		}

		totalSize += uint32(buf.Len())

		resp, err := c.rpcBus.Call(topics.SendMempoolTx, rpcbus.NewRequest(tx), 0)
		if err != nil {
			t.Fatal(err.Error())
		}
		txidBytes := resp.([]byte)

		payload, err := block.NewSHA3Payload(tx)
		txid, _ := payload.CalculateHash()
		if !bytes.Equal(txidBytes, txid) {
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
		c.addTx(tx)

		c.rpcBus.Call(topics.SendMempoolTx, rpcbus.NewRequest(tx), 5*time.Second)
	}

	// First, let's just get the entire view
	resp, err := c.m.SelectTx(context.Background(), &node.SelectRequest{})
	if err != nil {
		t.Fatal(err)
	}

	// There should be 16 txs
	assert.Equal(t, numTxs*4, len(resp.Result))

	// Now, we single out one hash from the bunch
	payload, err := block.NewSHA3Payload(txs[7])
	if err != nil {
		t.Fatal(err)
	}

	txid, _ := payload.CalculateHash()
	hash := hex.EncodeToString(txid)
	resp, err = c.m.SelectTx(context.Background(), &node.SelectRequest{Id: hash})
	if err != nil {
		t.Fatal(err)
	}

	// Should give us info about said tx
	assert.Equal(t, 1, len(resp.Result))
	if !strings.Contains(resp.Result[0].Id, hash) {
		t.Fatal("should have gotten info about requested tx")
	}

	// Let's filter for just stakes
	resp, err = c.m.SelectTx(context.Background(), &node.SelectRequest{Types: []node.TxType{node.TxType(2)}})
	if err != nil {
		t.Fatal(err)
	}

	// Should have `numTxs` lines, as there is one tx per type
	// per batch.
	assert.Equal(t, numTxs, len(resp.Result))
}

// Only difference with helper.RandomSliceOfTxs is lack of appending a coinbase tx
func randomSliceOfTxs(t *testing.T, txsBatchCount uint16) []transactions.ContractCall {
	var txs []transactions.ContractCall

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
