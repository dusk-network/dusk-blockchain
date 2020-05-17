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

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	assert "github.com/stretchr/testify/require"
)

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
	v := &transactions.MockProxy{}
	c.m = NewMempool(context.Background(), c.bus, c.rpcBus, v.Verifier(), v.Provider(), nil)
	c.m.Run()

	code := m.Run()
	os.Exit(code)
}

// adding tx in ctx means mempool is expected to store it in the verified list
func (c *ctx) addTx(tx transactions.ContractCall) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Distribute transactions should not get into the mempool
	if tx.Type() == transactions.Distribute {
		return
	}

	if err := c.m.verifier.VerifyTransaction(context.Background(), tx); err == nil {
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

	for i, tx := range c.verifiedTx {

		var exists bool
		for _, memTx := range txs {
			if transactions.Equal(memTx, tx) {
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

func prepTx(tx transactions.ContractCall) message.Message {
	return message.New(topics.Tx, tx)
}

// QUESTION: What does this test actually do?
func TestProcessPendingTxs(t *testing.T) {
	c.reset()

	cc := transactions.RandContractCalls(10, 0, false)

	for i := 0; i < 5; i++ {
		// Publish valid tx
		txMsg := prepTx(cc[i])
		c.addTx(cc[i])
		c.bus.Publish(topics.Tx, txMsg)

		// Publish invalid/valid txs (ones that do not pass verifyTx and ones that do)
		invalid := transactions.RandContractCall()
		transactions.Invalidate(invalid)
		txMsg = prepTx(invalid)
		c.addTx(invalid)
		c.bus.Publish(topics.Tx, txMsg)

		// Publish a duplicated tx
		c.addTx(invalid)
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
		txs := transactions.RandContractCalls(4, 0, false)
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
				tx := transactions.MockInvalidTx()
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
	assert := assert.New(t)

	c.reset()

	// Create a random block
	b := helper.RandomBlock(200, 0)
	b.Txs = make([]transactions.ContractCall, 0)

	counter := 0

	// generate 3*4 random txs
	txs := transactions.RandContractCalls(12, 0, false)
	for _, tx := range txs {
		// We avoid sharing this pointer between the mempool and the block
		// by marshaling and unmarshaling the tx
		buf := new(bytes.Buffer)
		assert.NoError(transactions.Marshal(buf, tx))

		txCopy, err := transactions.Unmarshal(buf)
		assert.NoError(err)

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

func TestCoinbaseTxsNotAllowed(t *testing.T) {
	c.reset()

	// Publish a set of valid txs and a Coinbase one
	txs := transactions.RandContractCalls(5, 0, true)

	for _, tx := range txs {
		txMsg := prepTx(tx)
		c.addTx(tx)
		c.bus.Publish(topics.Tx, txMsg)
	}

	c.wait()

	// Assert that all non-coinbase txs have been verified
	c.assert(t, true)
}

func TestSendMempoolTx(t *testing.T) {
	assert := assert.New(t)

	c.reset()

	txs := transactions.RandContractCalls(4, 0, false)

	var totalSize uint32
	for _, tx := range txs {
		buf := new(bytes.Buffer)
		assert.NoError(transactions.Marshal(buf, tx))

		totalSize += uint32(buf.Len())

		resp, err := c.rpcBus.Call(topics.SendMempoolTx, rpcbus.NewRequest(tx), 0)
		assert.NoError(err)
		txidBytes := resp.([]byte)

		txid, _ := tx.CalculateHash()
		assert.Equal(txidBytes, txid)
	}

	assert.Equal(c.m.verified.Size(), totalSize)
}

func TestMempoolView(t *testing.T) {
	assert := assert.New(t)
	c.reset()

	numTxs := 12
	txs := transactions.RandContractCalls(numTxs, 0, false)

	for _, tx := range txs {
		c.addTx(tx)
		c.rpcBus.Call(topics.SendMempoolTx, rpcbus.NewRequest(tx), 5*time.Second)
	}

	// First, let's just get the entire view
	resp, err := c.m.SelectTx(context.Background(), &node.SelectRequest{})
	assert.NoError(err)

	// There should be 12 txs
	assert.Equal(numTxs, len(resp.Result))

	// Now, we single out one hash from the bunch
	txid, _ := txs[7].CalculateHash()
	hash := hex.EncodeToString(txid)
	resp, err = c.m.SelectTx(context.Background(), &node.SelectRequest{Id: hash})
	assert.NoError(err)

	// Should give us info about said tx
	assert.Equal(1, len(resp.Result))
	// Checking that we got info about the Tx we actually requested
	assert.True(strings.Contains(resp.Result[0].Id, hash))

	// Let's test if the Select provides the right transactions
	bids := make([]*transactions.BidTransaction, 0)
	stakes := make([]*transactions.StakeTransaction, 0)
	stds := make([]*transactions.Transaction, 0)
	for i := 0; i < numTxs; i++ {
		switch txs[i].Type() {
		case transactions.Bid:
			bids = append(bids, txs[i].(*transactions.BidTransaction))
		case transactions.Stake:
			stakes = append(stakes, txs[i].(*transactions.StakeTransaction))
		case transactions.Tx:
			stds = append(stds, txs[i].(*transactions.Transaction))
		}
	}

	// check stakes
	resp, err = c.m.SelectTx(context.Background(), &node.SelectRequest{Types: []node.TxType{node.TxType_STAKE}})
	assert.NoError(err)
	assert.Equal(len(stakes), len(resp.Result))
	// check bids
	resp, err = c.m.SelectTx(context.Background(), &node.SelectRequest{Types: []node.TxType{node.TxType_BID}})
	assert.NoError(err)
	assert.Equal(len(bids), len(resp.Result))
	// check txs
	resp, err = c.m.SelectTx(context.Background(), &node.SelectRequest{Types: []node.TxType{node.TxType_STANDARD}})
	assert.NoError(err)
	assert.Equal(len(stds), len(resp.Result))
}
