package main

import (
	"bytes"
	"encoding/hex"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
	"testing"
	"time"
)

/**
 * This script creates a LevelDB blockchain database and fills it with 100 block headers.
 * Warning: It overwrites a possible existing 'test' directory under the root dir (dusk-go).
 */
func main() {
	createOneHundredBlocks()
}

func createOneHundredBlocks() {
	var blocks []*payload.Block
	var prevBlock = make([]byte, 32)

	marker := []byte("HasBeenInitialisedAlready")
	db, _ := database.NewBlockchainDB("test")
	db.Put(marker, []byte{})

	// First create the genesisblock
	genHdr, _ := createBlockFixture(0, prevBlock, 0)
	genesisHash, _ := hex.DecodeString("1ec2824a95be6188a6ffa51b3cfcfaacdcf09d07cdd46d1377e209318ba09bd5") // <= This is the genesis hash
	genHdr.Header.Hash = genesisHash

	blocks = append(blocks, genHdr)
	prevBlock = genHdr.Header.Hash

	for i := 1; i <= 100; i++ {
		block, _ := createBlockFixture(i, prevBlock, 0)
		prevBlock = block.Header.Hash
		blocks = append(blocks, block)
	}

	// AddHeaders
	hdrs := make([]*payload.BlockHeader, len(blocks))
	for i, block := range blocks {
		hdrs[i] = block.Header
	}
	db.AddHeaders(hdrs)
}

func createBlockFixture(height int, prevBlock []byte, txTotal int) (*payload.Block, error) {
	time := time.Now().Unix()
	// Spoof previous hash and seed
	//prevBlock, _ := crypto.RandEntropy(32)
	seed, _ := crypto.RandEntropy(32)
	txRoot, _ := crypto.RandEntropy(32)
	certImage, _ := crypto.RandEntropy(32)
	h := &payload.BlockHeader{uint64(height), time, prevBlock, seed, txRoot, nil, certImage}
	h.SetHash()

	// Create txTotal random Txs
	txs := createRandomTxFixtures(txTotal)

	return &payload.Block{h, txs}, nil
}

func createRandomTxFixtures(total int) []merkletree.Payload {
	txs := make([]merkletree.Payload, total)

	for i := 0; i < total; i++ {
		txID, _ := crypto.RandEntropy(32)
		sig, _ := crypto.RandEntropy(2000)
		in := &transactions.Input{TxID: txID, Index: 1, Signature: sig}
		dest, _ := crypto.RandEntropy(32)
		out := transactions.NewOutput(200, dest)
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

func createGenesisHeader(t *testing.T) {
	var b *payload.Block
	b, _ = createBlockFixture(0, make([]byte, 32), 0)

	layout := "2006-01-02T15:04:05.000Z"
	str := "2019-01-01T00:00:00.000Z"
	genTime, _ := time.Parse(layout, str)

	b.Header.Timestamp = genTime.Unix()
	b.SetHash()

	buf := new(bytes.Buffer)
	if err := b.Encode(buf); err != nil {
		t.Fatal(err)
	}
}
