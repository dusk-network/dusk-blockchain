// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package mempool

import (
	"bytes"
	"context"
	"math"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/sirupsen/logrus"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	assert "github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.FatalLevel)

	// config
	r := config.Registry{}
	r.Mempool.MaxSizeMB = 1
	r.Mempool.PoolType = "hashmap"
	r.Mempool.MaxInvItems = 10000
	config.Mock(&r)

	code := m.Run()
	os.Exit(code)
}

func startMempoolTest(ctx context.Context) (*Mempool, *eventbus.EventBus, *rpcbus.RPCBus, *eventbus.GossipStreamer) {
	return startMempoolTestWithLatency(ctx, time.Duration(0))
}

func startMempoolTestWithLatency(ctx context.Context, latency time.Duration) (*Mempool, *eventbus.EventBus, *rpcbus.RPCBus, *eventbus.GossipStreamer) {
	bus, streamer := eventbus.CreateGossipStreamer()

	rpcBus := rpcbus.New()
	v := &transactions.MockProxy{}
	m := NewMempool(nil, bus, rpcBus, v.ProberWithParams(latency), nil)

	m.Run(ctx)
	return m, bus, rpcBus, streamer
}

func TestTxAdvertising(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, _, _, streamer := startMempoolTest(ctx)

	tx := transactions.RandTx()

	go func() {
		_, err := m.ProcessTx("", message.New(topics.Tx, tx))
		assert.NoError(t, err)
	}()

	inv, err := streamer.Read()
	assert.NoError(t, err)

	hash, err := tx.CalculateHash()
	assert.NoError(t, err)

	msg := &message.Inv{}
	err = msg.Decode(bytes.NewBuffer(inv))
	assert.NoError(t, err)

	assert.Equal(t, msg.InvList[0].Hash, hash)
}

// QUESTION: What does this test actually do?
func TestProcessPendingTxs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, _, _, _ := startMempoolTest(ctx)

	cc := transactions.RandContractCalls(10, 0, false)

	for i := 0; i < 5; i++ {
		// Publish valid tx
		_, errList := m.ProcessTx("", message.New(topics.Tx, cc[i]))
		assert.Empty(t, errList)

		// Publish invalid/valid txs (ones that do not pass verifyTx and ones that do)
		invalid := transactions.RandContractCall()
		transactions.Invalidate(invalid)
		_, errList = m.ProcessTx("", message.New(topics.Tx, invalid))
		assert.NotEmpty(t, errList)

		// Publish a duplicated tx
		_, errList = m.ProcessTx("", message.New(topics.Tx, invalid))
		assert.NotEmpty(t, errList)
	}

	assert.Equal(t, m.verified.Len(), 5)

	for i := 0; i < 5; i++ {
		hash, err := cc[i].CalculateHash()
		assert.NoError(t, err)
		assert.True(t, m.verified.Contains(hash))
	}
}

func TestProcessPendingTxsAsync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, _, _, _ := startMempoolTest(ctx)

	// A batch consists of all 4 types of Dusk transactions (excluding coinbase)
	// The number of batches is the number of concurrent routines that will be
	// publishing the batch txs one by one. To avoid race conditions, as first
	// step we store here all txs that are expected to be in mempool later after
	// go-routines publishing
	batchCount := 8

	const numOfTxsPerBatch = 4

	txs := make([]transactions.ContractCall, 0)

	// generate and store txs that are expected to be valid
	for i := 0; i <= batchCount; i++ {
		// Generate a single batch of txs and added to the expected list of verified
		batchTxs := transactions.RandContractCalls(numOfTxsPerBatch, 0, false)
		txs = append(txs, batchTxs...)
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
				_, errList := m.ProcessTx("", message.New(topics.Tx, tx))
				assert.Empty(t, errList)
			}

			wg.Done()
		}(txs[from:to])
	}

	for i := 0; i < 2; i++ {
		wg.Add(1)
		// Publish invalid txs
		go func() {
			for j := 0; j <= 5; j++ {
				tx := transactions.MockInvalidTx()
				_, errList := m.ProcessTx("", message.New(topics.Tx, tx))
				assert.NotEmpty(t, errList)
			}
			wg.Done()
		}()
	}

	wg.Wait()

	for _, tx := range txs {
		hash, err := tx.CalculateHash()
		assert.NoError(t, err)
		assert.True(t, m.verified.Contains(hash))
	}
}

func TestRemoveAccepted(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, bus, rb, _ := startMempoolTest(ctx)

	// Create a random block
	b := helper.RandomBlock(200, 0)
	b.Txs = make([]transactions.ContractCall, 0)

	// generate 3*4 random txs
	txs := transactions.RandContractCalls(12, 0, false)
	for i, tx := range txs {
		// Publish valid tx
		// Copy the tx to avoid sharing pointers
		_, errList := m.ProcessTx("", message.New(topics.Tx, tx.Copy()))
		assert.Empty(errList)

		// Simulate a situation where the block has accepted each 2nd tx
		if math.Mod(float64(i), 2) == 0 {
			t := tx.(*transactions.Transaction)
			// If tx is accepted, it is expected to be removed from mempool on
			// onAcceptBlock event
			b.AddTx(t)
		}
	}

	root, _ := b.CalculateRoot()
	b.Header.TxRoot = root
	blockMsg := message.New(topics.AcceptedBlock, *b)
	errList := bus.Publish(topics.AcceptedBlock, blockMsg)
	assert.Empty(errList)

	time.Sleep(1 * time.Second)

	resp, err := rb.Call(topics.GetMempoolTxs, rpcbus.NewRequest(bytes.Buffer{}), 1*time.Second)
	assert.NoError(err)

	memTxs := resp.([]transactions.ContractCall)
	assert.Equal(len(memTxs), 6)

	for i, tx := range txs {
		hash, err := tx.CalculateHash()
		assert.NoError(err)

		if math.Mod(float64(i), 2) == 0 {
			assert.False(m.verified.Contains(hash))
		} else {
			assert.True(m.verified.Contains(hash))
		}
	}
}

func TestCoinbaseTxsNotAllowed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, _, rb, _ := startMempoolTest(ctx)

	// Publish a set of valid txs and a Coinbase one
	txs := transactions.RandContractCalls(5, 0, true)

	for _, tx := range txs {
		_, errList := m.ProcessTx("", message.New(topics.Tx, tx))

		if tx.Type() == transactions.Distribute {
			assert.NotEmpty(t, errList)
		}
	}

	// Assert that all non-coinbase txs have been verified
	resp, err := rb.Call(topics.GetMempoolTxs, rpcbus.NewRequest(bytes.Buffer{}), 1*time.Second)
	assert.NoError(t, err)

	memTxs := resp.([]transactions.ContractCall)
	for _, tx := range memTxs {
		assert.False(t, tx.Type() == transactions.Distribute)
	}
}

func TestSendMempoolTx(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, _, rb, _ := startMempoolTest(ctx)
	txs := transactions.RandContractCalls(4, 0, false)

	var totalSize uint32

	for _, tx := range txs {
		buf := new(bytes.Buffer)
		assert.NoError(transactions.Marshal(buf, tx))

		totalSize += uint32(buf.Len())

		resp, err := rb.Call(topics.SendMempoolTx, rpcbus.NewRequest(tx), 0)
		assert.NoError(err)

		txidBytes := resp.([]byte)

		txid, err := tx.CalculateHash()
		assert.Nil(err)

		assert.Equal(txidBytes, txid)
	}

	assert.Equal(m.verified.Size(), totalSize)
}

func BenchmarkProcessTx_0(b *testing.B) {
	// Recent result
	// BenchmarkProcessTx_0-8             50475             33671 ns/op
	// 33671 ns/op = 29699 TPS
	benchmarkProcessTx(b, 25000, 0)
}

func BenchmarkProcessTx_10(b *testing.B) {
	// Recent result
	// BenchmarkProcessTx_10-8              100          10346391 ns/op
	// 10346391 ns/op = 96 TPS
	benchmarkProcessTx(b, 1000, 10*time.Millisecond)
}

func BenchmarkProcessTx_20(b *testing.B) {
	// Recent result
	// BenchmarkProcessTx_20-8               58          20395738 ns/op
	// 20395738 ns/op = 58 TPS
	benchmarkProcessTx(b, 1000, 20*time.Millisecond)
}

//nolint
func benchmarkProcessTx(b *testing.B, batchCount int, verifyTransactionLatency time.Duration) {
	// test parameters
	const numOfTxsPerBatch = 4

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// NB VerifyTransaction has 0 latency
	m, _, _, _ := startMempoolTestWithLatency(ctx, verifyTransactionLatency)

	txs := make([]transactions.ContractCall, 0)

	for i := 0; i <= batchCount; i++ {
		// Generate a single batch of txs and added to the expected list of verified
		batchTxs := transactions.RandContractCalls(numOfTxsPerBatch, 0, false)
		txs = append(txs, batchTxs...)
	}

	b.ResetTimer()

	// Publish batchCount*numOfTxsPerBatch valid txs in a row
	var acceptedTxsCount int
	for n := 0; n < b.N; n++ {
		if n >= len(txs) {
			break
		}
		_, err := m.ProcessTx("unknown_addr", message.New(topics.Tx, txs[n]))
		if err != nil {
			b.Fatal(err)
		}

		acceptedTxsCount++
	}

	// Ensure all txs have been accepted
	if m.verified.Len() != acceptedTxsCount || acceptedTxsCount == 0 {
		b.Fatalf("not all txs accepted %d - %d", len(txs), m.verified.Len())
	}
}

/*
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
*/
