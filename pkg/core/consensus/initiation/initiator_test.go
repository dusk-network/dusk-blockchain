package initiation_test

import (
	"encoding/hex"
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/generation"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/initiation"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
	zkproof "github.com/dusk-network/dusk-zkproof"
	"github.com/stretchr/testify/assert"
)

// Test that the initiator functions properly when searching for a stake
func TestSearchForValueStake(t *testing.T) {
	keys, _ := user.NewRandKeys()
	db, err := initTest(t)
	assert.NoError(t, err)

	stake, err := helper.RandomStakeTx(t, false)
	assert.NoError(t, err)

	stake.PubKeyBLS = keys.BLSPubKeyBytes
	initiator := initiation.NewInitiator(db, stake.PubKeyBLS, consensus.FindStake)

	// We shouldn't get anything back yet, as the stake is not included in a block in the db
	_, err = initiator.SearchForValue()
	assert.Error(t, err)

	// Now, add the stake to the db
	storeTx(t, db, stake)

	// This time, we shouldn't get an error
	_, err = initiator.SearchForValue()
	assert.NoError(t, err)
}

func TestSearchForValueBid(t *testing.T) {
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

	initiator := initiation.NewInitiator(db, m.Bytes(), generation.FindD)
	// We shouldn't get anything back yet, as the bid is not included in a block in the db
	dBytes, err := initiator.SearchForValue()
	assert.Error(t, err, "was not supposed to get value back - got %s", hex.EncodeToString(dBytes))

	// Now, add the bid to the db
	storeTx(t, db, bid)

	// This time, we shouldn't get an error, and an actual value returned
	dBytes, err = initiator.SearchForValue()
	assert.NoError(t, err)
	assert.Equal(t, d.Bytes(), dBytes)
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
