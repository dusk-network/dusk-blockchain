package score

import (
	"context"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	assert "github.com/stretchr/testify/require"
)

func TestCorrectBidValues(t *testing.T) {
	assert := assert.New(t)
	eb := eventbus.New()
	keys, _ := key.NewRandKeys()
	_, db := lite.CreateDBConnection()

	p := transactions.MockProxy{}
	// FIXME: mock the BG here
	f := NewFactory(context.Background(), eb, keys, db, p)

	genesis := config.DecodeGenesis()
	assert.NoError(db.Update(func(t database.Transaction) error {
		return t.StoreBlock(genesis)
	}))

	// Add two sets of bid values, one expiring at 1000, and one
	// expiring at 2000
	d1, k1, edPk1, err := addBidValues(db, 1000)
	assert.NoError(err)
	d2, k2, edPk2, err := addBidValues(db, 2000)
	assert.NoError(err)

	c := f.Instantiate().(*Generator)

	assert.Equal(d1, c.d)
	assert.Equal(k1, c.k)
	assert.Equal(edPk1, c.edPk)

	// Now update our state so that the previous bid values are removed
	blk := helper.RandomBlock(1200, 1)
	assert.NoError(db.Update(func(t database.Transaction) error {
		return t.StoreBlock(blk)
	}))

	c = f.Instantiate().(*Generator)

	assert.Equal(d2, c.d)
	assert.Equal(k2, c.k)
	assert.Equal(edPk2, c.edPk)
}

func addBidValues(db database.DB, lockTime uint64) ([]byte, []byte, []byte, error) {
	d := transactions.Rand32Bytes()
	k := transactions.Rand32Bytes()
	edPk := transactions.Rand32Bytes()

	return d, k, edPk, db.Update(func(t database.Transaction) error {
		return t.StoreBidValues(d, k, edPk, lockTime)
	})
}
