package consensus_test

import (
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/generation"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
	zkproof "github.com/dusk-network/dusk-zkproof"
	"github.com/stretchr/testify/assert"
)

// Test that the initiator functions properly when searching for a stake
func TestSearchForStake(t *testing.T) {
	keys, _ := user.NewRandKeys()
	db, err := initTest(t)
	assert.NoError(t, err)

	stake, err := helper.RandomStakeTx(t, false)
	assert.NoError(t, err)

	stake.PubKeyBLS = keys.BLSPubKeyBytes
	retriever := consensus.NewTxRetriever(db, consensus.FindStake)

	// We shouldn't get anything back yet, as the stake is not included in a block in the db
	retrieved, err := retriever.SearchForTx(stake.PubKeyBLS)
	assert.Error(t, err)
	assert.Nil(t, retrieved)

	// Now, add the stake to the db
	storeTx(t, db, stake)

	// This time, we shouldn't get an error
	retrieved, err = retriever.SearchForTx(stake.PubKeyBLS)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.True(t, retrieved.Equals(stake))
}

func TestSearchForBid(t *testing.T) {
	k := ristretto.Scalar{}
	k.Rand()

	d := ristretto.Scalar{}
	d.Rand()
	db, err := initTest(t)
	assert.NoError(t, err)

	bid, err := helper.RandomBidTx(t, false)
	assert.NoError(t, err)

	bid.Outputs[0].Commitment = d.Bytes()
	m := zkproof.CalculateM(k)
	bid.M = m.Bytes()

	retriever := consensus.NewTxRetriever(db, generation.FindD)
	// We shouldn't get anything back yet, as the bid is not included in a block in the db
	retrieved, err := retriever.SearchForTx(m.Bytes())
	assert.Error(t, err)
	assert.Nil(t, retrieved)

	// Now, add the bid to the db
	storeTx(t, db, bid)

	// This time, we shouldn't get an error, and an actual value returned
	retrieved, err = retriever.SearchForTx(m.Bytes())
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, d.Bytes(), retrieved.(*transactions.Bid).Outputs[0].Commitment)
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
