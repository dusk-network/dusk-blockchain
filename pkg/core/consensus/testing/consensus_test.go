package testing

import (
	stdtesting "testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/score"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/stretchr/testify/assert"
)

type mockNode struct {
	hlp *score.Helper

	chain *mockChain

	// P2P Layer
	streamer eventbus.GossipStreamer
}

func (n *mockNode) boostrap(pubKey *keys.PublicKey, assert *assert.Assertions) {

	hlp := score.NewHelper(50, time.Second)
	// Subscribe for topics.Certificate

	streamer := eventbus.NewGossipStreamer(protocol.TestNet)
	streamListener := eventbus.NewStreamListener(streamer)
	// wiring the Gossip streamer to capture the gossiped messages
	_ = hlp.EventBus.Subscribe(topics.Gossip, streamListener)

	chain, err := newMockChain(*hlp.Emitter, 10*time.Second, pubKey, assert)
	assert.NoError(err)
	n.chain = chain

	n.chain.Loop(assert)
}

// TestConsensus performs a integration testing upon complete consensus logic
// TestConsensus passing means the consensus phases are properly assembled
func TestConsensus(t *stdtesting.T) {

	assert := assert.New(t)

	// boostrap node 1
	n := new(mockNode)

	_, pk := transactions.MockKeys()
	n.boostrap(pk, assert)

	time.Sleep(10 * time.Second)

	// TODO: assert chainTip is higher than prevChainTip
	// for each 5*second, fetch chainTip
	// assert chainTip is higher than prevChainTip

}
