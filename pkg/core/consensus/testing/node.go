package testing

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/score"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type mockNode struct {
	hlp   *score.Helper
	chain *mockChain
}

func newMockNode(pubKey *keys.PublicKey, gossipListener eventbus.Listener, assert *assert.Assertions) *mockNode {

	// Initialize hlp component
	hlp := score.NewHelper(50, time.Second)
	_ = hlp.EventBus.Subscribe(topics.Gossip, gossipListener)

	// Initialize Chain component
	chain, err := newMockChain(*hlp.Emitter, 10*time.Second, pubKey, assert)
	assert.NoError(err)

	return &mockNode{
		hlp:   hlp,
		chain: chain,
	}
}

func (n *mockNode) run(assert *assert.Assertions) {
	// Start Main Loop
	go n.chain.MainLoop(n.hlp.P, assert)
}

func (n *mockNode) getLastBlock() (block.Block, error) {
	req := rpcbus.NewRequest(nil)
	resp, err := n.hlp.RPCBus.Call(topics.GetLastBlock, req, time.Second)
	if err != nil {
		logrus.WithError(err).Error("timeout topics.GetLastBlock")
		return block.Block{}, err
	}
	blk := resp.(block.Block)
	return blk, nil
}
