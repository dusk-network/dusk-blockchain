package score

import (
	"testing"

	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/stretchr/testify/assert"
)

func TestCorrectBidValues(t *testing.T) {
	t.Skip("feature-419: Genesis block is broken. Unskip after moving to smart contract staking")
	eb := eventbus.New()
	keys, _ := key.NewRandKeys()
	_, db := lite.CreateDBConnection()

	f := NewFactory(eb, keys, db)

	genesis := config.DecodeGenesis()
	assert.NoError(t, db.Update(func(t database.Transaction) error {
		return t.StoreBlock(genesis)
	}))

	// Add two sets of bid values, one expiring at 1000, and one
	// expiring at 2000
	d1, k1, err := addBidValues(db, 1000)
	assert.NoError(t, err)
	d2, k2, err := addBidValues(db, 2000)
	assert.NoError(t, err)

	c := f.Instantiate().(*Generator)

	assert.Equal(t, d1, c.d.Bytes())
	assert.Equal(t, k1, c.k.Bytes())

	// Now update our state so that the previous bid values are removed
	blk := helper.RandomBlock(t, 1200, 1)
	assert.NoError(t, db.Update(func(t database.Transaction) error {
		return t.StoreBlock(blk)
	}))

	c = f.Instantiate().(*Generator)

	assert.Equal(t, d2, c.d.Bytes())
	assert.Equal(t, k2, c.k.Bytes())
}

//nolint:unused
func addBidValues(db database.DB, lockTime uint64) ([]byte, []byte, error) {
	var d ristretto.Scalar
	d.Rand()
	var k ristretto.Scalar
	k.Rand()
	return d.Bytes(), k.Bytes(), db.Update(func(t database.Transaction) error {
		return t.StoreBidValues(d.Bytes(), k.Bytes(), lockTime)
	})
}
