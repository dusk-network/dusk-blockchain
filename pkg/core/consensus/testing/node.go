// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package testing

import (
	"context"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/chain"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/loop"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
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

	pk := keys.PublicKey{
		AG: make([]byte, 32),
		BG: make([]byte, 32),
	}

	e := &consensus.Emitter{
		EventBus:    eb,
		RPCBus:      rb,
		Keys:        BLSKeys,
		TimerLength: 5 * time.Second,
	}
	lp := loop.New(e, &pk)

	c, err := chain.New(ctx, db, eb, rb, l, l, nil, proxy, lp)
	assert.NoError(err)
	return &node{chain: c}
}
