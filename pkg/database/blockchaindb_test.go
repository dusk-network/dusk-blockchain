package database_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

func TestAddHeader(t *testing.T) {
	db, _ := database.NewBlockchainDB(path)
	defer cleanup(db)

	block, _ := createBlockFixture()
	header := block.Header

	db.AddHeader(header)
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

func TestAddTransactions(t *testing.T) {
	db, _ := database.NewBlockchainDB(path)
	defer cleanup(db)

	b, err := createBlockFixture()
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

func createBlockFixture() (*payload.Block, error) {
	time := time.Now().Unix()
	// Spoof previous hash and seed
	prevBlock, _ := crypto.RandEntropy(32)
	seed, _ := crypto.RandEntropy(32)
	txRoot, _ := crypto.RandEntropy(32)
	hash, _ := crypto.RandEntropy(32)
	certImage, _ := crypto.RandEntropy(32)

	h := &payload.BlockHeader{1, time, prevBlock, seed, txRoot, hash, certImage}
	// Create 2 random Txs
	txs := createRandomTxFixtures(2)

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
