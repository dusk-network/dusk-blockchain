package testing

import (
	"context"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/chain"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/common"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/loop"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

// Node represents a stripped-down version of a Dusk blockchain node. It contains
// the minimum amount of components needed to function in consensus, and
// will be used to test the flow of the consensus and the chain.
type node struct {
	chain *chain.Chain
}

func newNode(ctx context.Context, assert *assert.Assertions, eb *eventbus.EventBus, rb *rpcbus.RPCBus, proxy transactions.Proxy, BLSKeys key.Keys) *node {
	_, db := lite.CreateDBConnection()

	// Just add genesis - we will fetch a different set of provisioners from
	// the `proxy` either way.
	genesis := config.DecodeGenesis()
	l := chain.NewDBLoader(db, genesis)
	_, err := l.LoadTip()
	assert.NoError(err)

	// Create some arbitrary bid values - they won't matter anyway, since
	// we are mocking proof verification.
	writeArbitraryBidValues(assert, db)

	pk := keys.PublicKey{
		AG: &common.JubJubCompressed{Data: make([]byte, 32)},
		BG: &common.JubJubCompressed{Data: make([]byte, 32)},
	}

	e := &consensus.Emitter{
		EventBus:    eb,
		RPCBus:      rb,
		Keys:        BLSKeys,
		Proxy:       proxy,
		TimerLength: 5 * time.Second,
	}
	lp := loop.New(e, &pk)

	c, err := chain.New(ctx, db, eb, rb, l, l, nil, proxy, lp)
	assert.NoError(err)
	go func() {
		if err := c.ProduceBlock(ctx); err != nil && err != context.Canceled {
			panic(err)
		}
	}()

	return &node{chain: c}
}

func writeArbitraryBidValues(assert *assert.Assertions, db database.DB) {
	d, err := crypto.RandEntropy(32)
	assert.NoError(err)
	k, err := crypto.RandEntropy(32)
	assert.NoError(err)
	index := uint64(0)
	lockTime := uint64(250000)
	assert.NoError(db.Update(func(t database.Transaction) error {
		return t.StoreBidValues(d, k, index, lockTime)
	}))
}
