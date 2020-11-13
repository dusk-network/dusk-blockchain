package testing

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

//nolint:unused
type mockNode struct {
	chain *mockChain
	*consensus.Emitter

	ThisSender       []byte
	ProvisionersKeys []key.Keys
	P                *user.Provisioners
}

//nolint:unused
func newMockNode(pubKey *keys.PublicKey, provisionersKeys []key.Keys,
	p *user.Provisioners, index int, gossipListener eventbus.Listener,
	genesis block.Block, lastCert block.Certificate,
	assert *assert.Assertions) *mockNode {

	mockProxy := transactions.MockProxy{
		P:  transactions.PermissiveProvisioner{},
		BG: transactions.MockBlockGenerator{},
	}

	emitter := consensus.MockEmitter(10*time.Second, mockProxy)
	emitter.Keys = provisionersKeys[index]

	// Initialize hlp component
	_ = emitter.EventBus.Subscribe(topics.Gossip, gossipListener)

	// Initialize Chain component
	chain, err := newMockChain(*emitter, 10*time.Second, pubKey, genesis, lastCert, assert)
	assert.NoError(err)

	return &mockNode{
		chain:            chain,
		ThisSender:       emitter.Keys.BLSPubKeyBytes,
		ProvisionersKeys: provisionersKeys,
		P:                p,
		Emitter:          emitter,
	}
}

func (n *mockNode) run(assert *assert.Assertions) {
	// Start Main Loop
	go n.chain.MainLoop(n.P, assert)
}

func (n *mockNode) getLastBlock() (block.Block, error) {
	req := rpcbus.NewRequest(nil)
	resp, err := n.RPCBus.Call(topics.GetLastBlock, req, time.Second)
	if err != nil {
		logrus.WithError(err).Error("timeout topics.GetLastBlock")
		return block.Block{}, err
	}
	blk := resp.(block.Block)
	return blk, nil
}
