package database_test

import (
	"bytes"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/util"
	"io/ioutil"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

func init() {
	log.SetOutput(ioutil.Discard)
}

func TestAddHeader(t *testing.T) {
	db, _ := database.NewBlockchainDB(path)
	defer cleanup(db)

	blocks, _ := createBlockFixtures(1, 0)
	header := blocks[0].Header

	db.AddHeaders([]*payload.BlockHeader{header})
	assert.NotEqual(t, nil, db)

	heightBytes := database.Uint64ToBytes(header.Height)
	heightKey := append(database.BLOCKHEIGHT, heightBytes...)
	dbHeaderHash, _ := db.Get(heightKey)

	headerKey := append(database.HEADER, dbHeaderHash...)
	dbHeaderBytes, _ := db.Get(headerKey)
	headerBytes, _ := header.Bytes()

	dbLatestHdr, _ := db.Get(database.LATESTHEADER)

	assert.Equal(t, header.Hash, dbHeaderHash)
	assert.Equal(t, headerBytes, dbHeaderBytes)
	assert.Equal(t, header.Hash, dbLatestHdr)
}

func TestAddHeaders(t *testing.T) {
	db, _ := database.NewBlockchainDB(path)
	defer cleanup(db)

	blocks, _ := createBlockFixtures(5, 0)

	hdrs := make([]*payload.BlockHeader, len(blocks))
	for i, block := range blocks {
		hdrs[i] = block.Header
	}

	db.AddHeaders(hdrs)

	assert.NotEqual(t, nil, db)

	for _, hdr := range hdrs {
		headerKey := append(database.HEADER, hdr.Hash...)
		dbHeaderBytes, _ := db.Get(headerKey)
		hdrBytes, _ := hdr.Bytes()
		assert.Equal(t, hdrBytes, dbHeaderBytes)
	}

	// Assert LATESTHEADER really has the right height
	sortedHdrs := util.SortHeadersByHeight(hdrs)
	latestHeight := sortedHdrs[len(sortedHdrs)-1].Height
	latestHash := sortedHdrs[len(sortedHdrs)-1].Hash

	heightBytes := database.Uint64ToBytes(latestHeight)
	heightKey := append(database.BLOCKHEIGHT, heightBytes...)
	dbLatestHeightHash, _ := db.Get(heightKey)
	dbLatestHeaderHash, _ := db.Get(database.LATESTHEADER)

	assert.Equal(t, latestHash, dbLatestHeightHash)
	assert.Equal(t, dbLatestHeightHash, dbLatestHeaderHash)
}

func TestAddTransactions(t *testing.T) {
	db, _ := database.NewBlockchainDB(path)
	defer cleanup(db)

	blocks, err := createBlockFixtures(1, 2)
	b := blocks[0]
	if err != nil {
		t.FailNow()
	}
	h := b.Header

	stealths := make([]transactions.Stealth, len(b.Txs))
	for i, tx := range b.Txs {
		stealths[i] = *tx.(*transactions.Stealth)
	}
	db.AddTransactions(h.Hash, stealths)

	// Run through the txs
	for i, tx := range stealths {
		buf := new(bytes.Buffer)
		tx.Encode(buf)
		txBytes := buf.Bytes()

		txKey := append(database.TX, tx.Hash...)
		dbTxBytes, _ := db.Get(txKey)

		blockhashKey := append(database.TX, h.Hash...)
		blockhashKey = append(blockhashKey, database.Uint32ToBytes(uint32(i))...)
		dbTxHash, _ := db.Get(blockhashKey)

		assert.Equal(t, txBytes, dbTxBytes)
		assert.Equal(t, tx.Hash, dbTxHash)
	}
}

func createBlockFixtures(totalBlocks, totalTxs int) ([]*payload.Block, error) {
	blocks := make([]*payload.Block, totalBlocks)
	time := time.Now().Unix()
	// Spoof previous hash and seed
	prevBlock, _ := crypto.RandEntropy(32)
	seed, _ := crypto.RandEntropy(32)
	txRoot, _ := crypto.RandEntropy(32)
	certImage, _ := crypto.RandEntropy(32)

	for i := 0; i < totalBlocks; i++ {
		h := &payload.BlockHeader{uint64(i), time, prevBlock, seed, txRoot, nil, certImage}
		h.SetHash()
		// Create random Txs
		txs := createRandomTxFixtures(totalTxs)
		blocks[i] = &payload.Block{h, txs}
	}

	return blocks, nil
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
