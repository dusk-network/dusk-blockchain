package generation

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/bwesterb/go-ristretto"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/core/mempool"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet/transactions"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-wallet/key"
)

func respond(rpcBus *rpcbus.RPCBus, b block.Block) {
	r := <-rpcbus.GetLastBlockChan
	buf := new(bytes.Buffer)
	if err := block.Marshal(buf, &b); err != nil {
		panic(err)
	}
	r.RespChan <- *buf
}

func TestGenerateBlock(t *testing.T) {
	h := newTestHarness(t)
	defer h.close()

	// Publish a few random txs
	txsCount, err := publishRandomTxs(t, h)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(500 * time.Millisecond)

	keys, _ := user.NewRandKeys()
	// Block Generator to construct a valid block
	gen := newBlockGenerator(h.genWallet.PublicKey(), h.rpc, nil, keys)

	seed, _ := crypto.RandEntropy(33)
	proof, _ := crypto.RandEntropy(32)
	score, _ := crypto.RandEntropy(32)

	var prevBlockRound uint64 = 2
	prevBlock := *helper.RandomBlock(t, prevBlockRound, 1)

	round := prevBlockRound + 1

	// Generate a candidate block based on the mocked states of mempool and
	// prevBlock
	candidateBlock, err := gen.GenerateBlock(round, seed, proof, score, prevBlock.Header.Hash)
	if err != nil {
		t.Fatalf("GenerateBlock returned err %s", err.Error())
	}

	// As GenerateBlock internally verifies the candidate block, if no error is
	// returned we can assume it's a valid block.
	if candidateBlock.Header.Height != round {
		t.Fatalf("expecting candidate block to be the target round")
	}

	if !bytes.Equal(candidateBlock.Header.Seed, seed) {
		t.Fatalf("expecting candidate block to store the seed value properly")
	}

	if len(candidateBlock.Txs) != txsCount+1 {
		t.Fatalf("expecting candidate block to include all mempool txs plus the coinbase tx")
	}

	coinbaseTx := candidateBlock.Txs[0].(*transactions.Coinbase)

	// Verify if block generator is capable of spending the constructed coinbase tx
	if !canSpend(t, coinbaseTx, h) {
		t.Fatalf("block generator cannot spend this coinbase reward output")
	}

	// Ensure proof and score are also stored
	if !bytes.Equal(coinbaseTx.Proof, proof) {
		t.Fatalf("expecting candidate block to store the proof value properly")
	}

	if !bytes.Equal(coinbaseTx.Score, score) {
		t.Fatalf("expecting candidate block to store the proof value properly")
	}
}

func publishRandomTxs(t *testing.T, h *harness) (int, error) {
	txs := helper.RandomSliceOfTxs(t, 1)
	txsCount := 0
	for _, tx := range txs {

		if tx.Type() == transactions.CoinbaseType {
			continue
		}

		// Publish non-coinbase tx
		buf := new(bytes.Buffer)
		err := transactions.Marshal(buf, tx)
		if err != nil {
			return 0, err
		}

		h.eb.Publish(string(topics.Tx), *buf)
		txsCount++
	}

	return txsCount, nil
}

func canSpend(t *testing.T, tx *transactions.Coinbase, h *harness) bool {
	P := tx.Rewards[0].PubKey.P
	_, spendable := h.genWallet.DidReceiveTx(tx.R, key.StealthAddress{P: P}, 0)
	return spendable
}

func bytesToPoint(b []byte) ristretto.Point {
	var buf [32]byte
	copy(buf[:], b[:])

	var p ristretto.Point
	_ = p.SetBytes(&buf)
	return p
}

// TODO: this can be moved eventually to a separate pkg to be reused by all subsystems
type harness struct {
	tmpDataDir string

	eb  *eventbus.EventBus
	rpc *rpcbus.RPCBus
	m   *mempool.Mempool

	// block generator tmp wallet
	genWallet *key.Key
}

func newTestHarness(t *testing.T) *harness {

	h := &harness{}

	tmpDataDir, err := ioutil.TempDir(os.TempDir(), "chain_test")
	if err != nil {
		t.Fatal(err.Error())
	}

	// Mock configuration
	r := cfg.Registry{}
	r.Database.Dir = tmpDataDir
	r.Database.Driver = heavy.DriverName
	r.General.Network = "testnet"

	r.Mempool.MaxSizeMB = 1
	r.Mempool.PoolType = "hashmap"

	cfg.Mock(&r)

	// Mock event bus object
	h.eb = eventbus.New()
	h.rpc = rpcbus.New()

	// Mock mempool with no verification procedure
	h.m = mempool.NewMempool(h.eb, func(tx transactions.Transaction) error {
		return nil
	})

	h.m.Run()

	// Generator keys
	seed := make([]byte, 64)

	_, err = rand.Read(seed)
	if err != nil {
		t.Fatal(err.Error())
	}

	h.genWallet = key.NewKeyPair(seed)
	return h
}

func (h *harness) close() {
	os.RemoveAll(h.tmpDataDir)
}
