package main

import (
	"encoding/hex"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/noded/logging"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	userHomeDuskDir = "/.dusk"
)

var marker = []byte("HasBeenInitialisedAlready")

/**
 * This script creates a LevelDB blockchain database and fills it with block headers and blocks.
 * Parameters: action. env, nrofblocks, txperblock
 * Examples:
 * Initialize a blockchain db with 1000 blocks with each 3 transactions:
 * 		go run AddToBlockchainDb.go init devnet 1000 3
 * Add 2000 blocks to an existing blockchain db with each 2 transactions:
 * 		go run AddToBlockchainDb.go add devnet 2000 2
 * Warning: Action 'init' overwrites an existing db directory (<user home>/.dusk/<env>/db).
 */

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{DisableColors: true})
}

func main() {
	args := os.Args[1:]
	action := args[0]
	env := args[1]
	totBlocks, _ := strconv.ParseUint(args[2], 10, 32)
	txsPerBlock, _ := strconv.ParseUint(args[3], 10, 32)
	if action == "init" {
		createBlocks(env, int(totBlocks), int(txsPerBlock))
	}
	if action == "add" {
		addBlocks(env, int(totBlocks), int(txsPerBlock))
	}
}

func createBlocks(env string, totBlocks int, txsPerBlock int) {
	var prevHash = make([]byte, 32)
	blocks := make([]*payload.Block, 0, totBlocks)

	path := logging.UserHomeDir() + userHomeDuskDir + "/" + strings.ToLower(env) + "/db"
	db, _ := database.NewBlockchainDB(path)
	db.Put(marker, []byte{})

	// First create the Genesis block
	genBlk, _ := createBlockFixture(0, prevHash, 0)
	genesisHash, _ := hex.DecodeString("1ec2824a95be6188a6ffa51b3cfcfaacdcf09d07cdd46d1377e209318ba09bd5") // <= This is the genesis hash
	genBlk.Header.Hash = genesisHash

	blocks = append(blocks, genBlk)
	prevHash = genBlk.Header.Hash

	for i := 1; i <= totBlocks; i++ {
		block, _ := createBlockFixture(uint64(i), prevHash, txsPerBlock)
		prevHash = block.Header.Hash
		blocks = append(blocks, block)
	}

	// WriteHeaders
	hdrs := make([]*payload.BlockHeader, len(blocks))
	for i, block := range blocks {
		hdrs[i] = block.Header
	}
	db.WriteHeaders(hdrs)
	db.WriteBlockTransactions(blocks)
}

func addBlocks(env string, totBlocks int, txsPerBlock int) {
	blocks := make([]*payload.Block, 0, totBlocks)
	// Read last header
	path := logging.UserHomeDir() + userHomeDuskDir + "/" + strings.ToLower(env) + "/db"

	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("Action 'add' can only be used with an existing blockchain db.")
		return
	}
	db, _ := database.NewBlockchainDB(path)

	// Check if existing db
	init, _ := db.Has(marker)
	if !init {
		fmt.Println("Action 'add' can only be used with an initialized blockchain db.")
		fmt.Println("Use Action 'init' first to initialize a blockchain db.")

	}

	latestHdr, _ := db.GetLatestHeader()
	prevHash := latestHdr.Hash

	for i := 1; i <= totBlocks; i++ {
		block, _ := createBlockFixture(latestHdr.Height+uint64(i), prevHash, txsPerBlock)
		prevHash = block.Header.Hash
		blocks = append(blocks, block)
	}

	// WriteHeaders
	hdrs := make([]*payload.BlockHeader, len(blocks))
	for i, block := range blocks {
		hdrs[i] = block.Header
	}
	db.WriteHeaders(hdrs)
	db.WriteBlockTransactions(blocks)
}

func createBlockFixture(height uint64, prevBlock []byte, txTotal int) (*payload.Block, error) {
	t := time.Now().Unix()
	// Spoof previous seed, txRoot and certImage
	seed, _ := crypto.RandEntropy(32)
	certImage, _ := crypto.RandEntropy(32)
	h := &payload.BlockHeader{Height: uint64(height), Timestamp: t, PrevBlock: prevBlock, Seed: seed, Hash: nil, CertImage: certImage}

	// Create txTotal random Txs
	txs := createRandomTxFixtures(txTotal)
	b := &payload.Block{h, txs}

	// Create txRoot
	if len(b.Txs) > 0 {
		tree, _ := merkletree.NewTree(b.Txs)
		b.Header.TxRoot = tree.MerkleRoot
	} else {
		b.Header.TxRoot = make([]byte, 32)
	}
	h.SetHash()

	return b, nil
}

func createRandomTxFixtures(total int) []merkletree.Payload {
	txs := make([]merkletree.Payload, total)

	if len(txs) < 1 {
		return txs
	}

	for i := 0; i < total; i++ {
		keyImage, _ := crypto.RandEntropy(32)
		txID, _ := crypto.RandEntropy(32)
		sig, _ := crypto.RandEntropy(2000)
		in := &transactions.Input{KeyImage: keyImage, TxID: txID, Index: uint8(i), Signature: sig}
		dest, _ := crypto.RandEntropy(32)

		amount := rand.Intn(1000) + 1
		out := transactions.NewOutput(uint64(amount), dest, sig)

		txPubKey, _ := crypto.RandEntropy(32)
		s := transactions.NewTX()
		s.AddInput(in)
		s.AddOutput(out)
		s.AddTxPubKey(txPubKey)
		s.SetHash()
		txs[i] = s
	}
	return txs
}
