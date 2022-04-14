// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package testing

import (
	"os"
	"strconv"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/stretchr/testify/assert"
)

// Block generators will need to communicate with the mempool. So we can
// mock a collector here instead, which just provides them with nothing.
func catchGetMempoolTxsBySize(assert *assert.Assertions, rb *rpcbus.RPCBus) {
	c := make(chan rpcbus.Request, 20)
	assert.NoError(rb.Register(topics.GetMempoolTxsBySize, c))

	go func() {
		for {
			r := <-c
			r.RespChan <- rpcbus.Response{
				Resp: []transactions.ContractCall{transactions.RandTx()},
				Err:  nil,
			}
		}
	}()
}

func setupProvisioners(assert *assert.Assertions, amount int) (user.Provisioners, []key.Keys) {
	p := user.NewProvisioners()
	keys := make([]key.Keys, amount)

	for i := 0; i < amount; i++ {
		keys[i] = key.NewRandKeys()

		// All nodes are given an equal stake and locktime
		assert.NoError(p.Add(keys[i].BLSPubKey, 0, 100000, 0, 250000))
	}

	return *p, keys
}

func mockProxy(p user.Provisioners) transactions.Proxy {
	return transactions.MockProxy{
		E: &transactions.PermissiveExecutor{
			P: &p,
		},
	}
}

type gossipRouter struct {
	eb eventbus.Publisher
	d  *dupemap.DupeMap
}

func (g *gossipRouter) route(m message.Message) {
	b := m.Payload().(message.SafeBuffer).Buffer
	if g.d.HasAnywhere(&b) {
		// The incoming message will be a message.SafeBuffer, as it's coming from
		// the consensus.Emitter.
		m, err := message.Unmarshal(&b, nil)
		if err != nil {
			panic(err)
		}

		go g.eb.Publish(m.Category(), m)
	}
}

func rerouteGossip(eb *eventbus.EventBus) {
	router := &gossipRouter{eb, dupemap.NewDupeMap(5, 100)}
	gossipListener := eventbus.NewSafeCallbackListener(router.route)
	eb.Subscribe(topics.Gossip, gossipListener)
}

func getNumNodes(assert *assert.Assertions) int {
	numNodesStr := os.Getenv("DUSK_TESTBED_NUM_NODES")
	if numNodesStr != "" {
		numNodes, err := strconv.Atoi(numNodesStr)
		assert.NoError(err)
		return numNodes
	}

	// Standard test case runs with 10 nodes.
	return 10
}

func getNumRounds(assert *assert.Assertions) int {
	numRoundsStr := os.Getenv("DUSK_TESTBED_NUM_ROUNDS")
	if numRoundsStr != "" {
		numRounds, err := strconv.Atoi(numRoundsStr)
		assert.NoError(err)
		return numRounds
	}

	// Standard test case runs for 10 rounds.
	return 10
}
