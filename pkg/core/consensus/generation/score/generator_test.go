package score

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/stretchr/testify/assert"
)

func TestCorrectBidValues(t *testing.T) {
	eb := eventbus.New()
	keys, _ := key.NewRandKeys()
	_, db := lite.CreateDBConnection()

	p := transactions.MockProxy{}
	// FIXME: mock the BG here
	f := NewFactory(context.Background(), eb, keys, db, p)

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

	assert.Equal(t, d1, c.d)
	assert.Equal(t, k1, c.k)

	// Now update our state so that the previous bid values are removed
	blk := helper.RandomBlock(1200, 1)
	assert.NoError(t, db.Update(func(t database.Transaction) error {
		return t.StoreBlock(blk)
	}))

	c = f.Instantiate().(*Generator)

	assert.Equal(t, d2, c.d)
	assert.Equal(t, k2, c.k)
}

func addBidValues(db database.DB, lockTime uint64) ([]byte, []byte, error) {
	d := make([]byte, 64)
	k := make([]byte, 64)
	_, _ = rand.Read(d)
	_, _ = rand.Read(k)

	return d, k, db.Update(func(t database.Transaction) error {
		return t.StoreBidValues(d, k, lockTime)
	})
}
