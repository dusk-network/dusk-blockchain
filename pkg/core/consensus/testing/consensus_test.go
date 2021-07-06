// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package testing

import (
	"context"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/stretchr/testify/assert"
)

// This is a broad integration test, which spins up a given number of nodes
// and makes them perform consensus with each other. We then observe that
// all results are as expected.
//
// The test does not make use of the network, and all nodes communicate
// over a single event bus. This removes the network-based asynchronism
// from the test case, and allows us to focus solely on consensus flow
// and possible corner cases.
func TestConsensus(t *testing.T) {
	assert := assert.New(t)

	// logrus.SetLevel(logrus.DebugLevel)

	// Retrieve the amount of nodes to use for this test case.
	numNodes := getNumNodes(assert)

	// Retrieve the amount of rounds to run this test case for.
	numRounds := getNumRounds(assert)

	// Create a context from which we can cancel the whole testbed.
	ctx, cancel := context.WithCancel(context.Background())
	// Ensure we clean up at the end of this test.
	defer cancel()

	eb, rb := eventbus.New(), rpcbus.New()

	// We need to provide block generators with transactions.
	go catchGetMempoolTxsBySize(assert, rb)

	// We should listen to `AcceptedBlock` messages, so that we can keep
	// an overview of where the consensus is.
	abChan := make(chan message.Message, numNodes)
	abListener := eventbus.NewChanListener(abChan)
	eb.Subscribe(topics.AcceptedBlock, abListener)

	p, keys := setupProvisioners(assert, numNodes)
	proxy := mockProxy(p)

	// Gossip messages need to be rerouted back to the event bus,
	// so that all nodes receive them.
	rerouteGossip(eb)

	// Create a set of nodes
	nodes := make([]*node, numNodes)

	for i := 0; i < numNodes; i++ {
		nctx, cancelNode := context.WithCancel(ctx)
		nodes[i] = newNode(nctx, assert, eb, rb, proxy, keys[i])
		// Resource clean up.
		defer cancelNode()
	}

	// Start the consensus loop on all nodes
	for _, n := range nodes {
		go func(n *node) {
			if err := n.chain.ProduceBlock(); err != nil && err != context.Canceled {
				panic(err)
			}
		}(n)
	}

	// To follow consensus properly, we should see accepted blocks coming in
	// from each node.
	// Let's make sure the consensus can survive for at least up to 10 rounds.
	for i := 1; i <= numRounds; i++ {
		counter := 0

		for {
			b := <-abChan
			if b.Payload().(block.Block).Header.Height != uint64(i) {
				t.Fatal("consensus was desynchronized")
			}

			counter++

			// Once we get the same block `numNodes` times, we can expect the blocks
			// of the next round to come in.
			if counter == numNodes {
				break
			}
		}
	}
}
