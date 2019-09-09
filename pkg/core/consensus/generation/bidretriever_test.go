package generation

import (
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet/transactions"
	zkproof "github.com/dusk-network/dusk-zkproof"
	"github.com/stretchr/testify/assert"
)

func TestSearchForBid(t *testing.T) {
	k := ristretto.Scalar{}
	k.Rand()

	d := ristretto.Point{}
	d.Rand()
	db, err := initTest(t)
	assert.NoError(t, err)

	bid, err := helper.RandomBidTx(t, false)
	assert.NoError(t, err)

	bid.Outputs[0].Commitment = d
	m := zkproof.CalculateM(k)
	bid.M = m.Bytes()

	retriever := newBidRetriever(db)
	// We shouldn't get anything back yet, as the bid is not included in a block in the db
	retrieved, err := retriever.SearchForBid(m.Bytes())
	assert.Error(t, err)
	assert.Nil(t, retrieved)

	// Now, add the bid to the db
	storeTx(t, db, bid)

	// This time, we shouldn't get an error, and an actual value returned
	retrieved, err = retriever.SearchForBid(m.Bytes())
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, d.Bytes(), retrieved.(*transactions.Bid).Outputs[0].Commitment.Bytes())
}

func initTest(t *testing.T) (database.DB, error) {
	_, db := lite.CreateDBConnection()
	// Add a genesis block so we don't run into any panics
	blk := helper.RandomBlock(t, 0, 2)
	return db, db.Update(func(t database.Transaction) error {
		return t.StoreBlock(blk)
	})
}

func storeTx(t *testing.T, db database.DB, tx transactions.Transaction) {
	blk := helper.RandomBlock(t, 1, 2)
	blk.AddTx(tx)
	err := db.Update(func(t database.Transaction) error {
		return t.StoreBlock(blk)
	})
	assert.NoError(t, err)
}
