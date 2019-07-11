package generation

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/bwesterb/go-ristretto"
	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/heavy"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/mempool"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/key"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func respond(rpcBus *wire.RPCBus, b block.Block) {
	r := <-wire.GetLastBlockChan
	buf := new(bytes.Buffer)
	if err := b.Encode(buf); err != nil {
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

	// Block Generator to construct a valid block
	gen := newBlockGenerator(h.genWallet.PublicKey(), h.rpc)

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
		err := tx.Encode(buf)
		if err != nil {
			return 0, err
		}

		h.eb.Publish(string(topics.Tx), buf)
		txsCount++
	}

	return txsCount, nil
}

func canSpend(t *testing.T, tx *transactions.Coinbase, h *harness) bool {
	P := bytesToPoint(tx.Rewards[0].DestKey)
	R := bytesToPoint(tx.R[:])
	_, spendable := h.genWallet.DidReceiveTx(R, key.StealthAddress{P: P}, 0)
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

	eb  *wire.EventBus
	rpc *wire.RPCBus
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
	h.eb = wire.NewEventBus()
	h.rpc = wire.NewRPCBus()

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
