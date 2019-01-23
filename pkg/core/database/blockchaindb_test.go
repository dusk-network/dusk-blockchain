package database_test

import (
	"bytes"
	"io/ioutil"
	"testing"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/util"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
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

	db.WriteHeaders([]*block.Header{header})
	assert.NotEqual(t, nil, db)

	heightBytes := database.Uint64ToBytes(header.Height)
	table := database.NewTable(db, append(database.BLOCKHEIGHT, heightBytes...))
	heightKey := append(database.BLOCKHEIGHT, heightBytes...)

	table = database.NewTable(db, database.HEADER)
	dbHeaderHash, _ := db.Get(heightKey)

	table = database.NewTable(db, database.HEADER)
	dbHeaderBytes, _ := table.Get(dbHeaderHash)

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

	hdrs := make([]*block.Header, len(blocks))
	for i, block := range blocks {
		hdrs[i] = block.Header
	}

	db.WriteHeaders(hdrs)

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

func TestAddBlockTransactions(t *testing.T) {
	db, _ := database.NewBlockchainDB(path)
	defer cleanup(db)

	blocks, err := createBlockFixtures(2, 2)
	if err != nil {
		t.FailNow()
	}

	db.WriteBlockTransactions(blocks)

	// Run through the txs and assert them
	for _, b := range blocks {
		for i, v := range b.Txs {
			tx := v.(*transactions.Stealth)
			buf := new(bytes.Buffer)
			tx.Encode(buf)
			txBytes := buf.Bytes()

			txKey := append(database.TX, tx.Hash...)
			dbTxBytes, _ := db.Get(txKey)

			txHashKey := append(database.TX, b.Header.Hash...)
			txHashKey = append(txHashKey, database.Uint32ToBytes(uint32(i))...)
			dbTxHash, _ := db.Get(txHashKey)

			assert.Equal(t, txBytes, dbTxBytes)
			assert.Equal(t, tx.Hash, dbTxHash)
		}
	}
}

func createBlockFixtures(totalBlocks, totalTxs int) ([]*block.Block, error) {
	blocks := make([]*block.Block, totalBlocks)
	time := time.Now().Unix()
	// Spoof previous hash and seed
	prevBlock, _ := crypto.RandEntropy(32)
	seed, _ := crypto.RandEntropy(32)
	txRoot, _ := crypto.RandEntropy(32)
	certImage, _ := crypto.RandEntropy(32)

	for i := 0; i < totalBlocks; i++ {
		h := &block.Header{
			Height:    uint64(i),
			Timestamp: time,
			PrevBlock: prevBlock,
			Seed:      seed,
			TxRoot:    txRoot,
			Hash:      nil,
			CertImage: certImage,
		}
		h.SetHash()
		// Create random Txs
		txs := createRandomTxFixtures(totalTxs)
		blocks[i] = &block.Block{
			Header: h,
			Txs:    txs,
		}
	}

	return blocks, nil
}

func createRandomTxFixtures(total int) []merkletree.Payload {
	txs := make([]merkletree.Payload, total)

	for i := 0; i < total; i++ {
		txID, _ := crypto.RandEntropy(32)
		sig, _ := crypto.RandEntropy(2000)
		key, _ := crypto.RandEntropy(32)
		in := transactions.NewInput(key, txID, 1, sig)
		dest, _ := crypto.RandEntropy(32)
		out := transactions.NewOutput(200, dest, sig)
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
