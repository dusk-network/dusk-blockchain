package testing

import (
	"encoding/hex"
	stdtesting "testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// nolint
func mockGenesis() (block.Block, block.Certificate) {
	randomGenesis := helper.RandomBlock(0, 3)
	lastCertificate := helper.RandomCertificate()
	randomGenesis.Header.Timestamp = time.Now().Unix() - 100000
	return *randomGenesis, *lastCertificate
}

// TestConsensus performs a integration testing upon complete consensus logic
// TestConsensus passing means the consensus phases are properly assembled
func TestConsensus(t *stdtesting.T) {

	// logrus.SetLevel(logrus.TraceLevel)

	assert := assert.New(t)

	// Create Gossip Router
	streamer := eventbus.NewRouterStreamer()
	streamListener := eventbus.NewStreamListener(streamer)

	network := make([]mockNode, 0)
	networkSize := 3

	// Mock provisioners
	provisioners := networkSize
	p, provisionersKeys := consensus.MockProvisioners(provisioners)

	// Mock genesis
	genesis, cert := mockGenesis()

	// Initialize consensus participants
	for i := 0; i < networkSize; i++ {
		_, pk := transactions.MockKeys()

		node := newMockNode(pk, provisionersKeys, p, i, streamListener, genesis, cert, assert)

		network = append(network, *node)
		streamer.Add(node.EventBus)
	}

	// Run all consensus participants
	for n := 0; n < len(network); n++ {
		network[n].run(assert)
	}

	// Monitor consensus participants
	for {
		time.Sleep(7 * time.Second)

		h := uint64(0)
		for i := 0; i < len(network); i++ {

			// Trace chain tip of all nodes
			blk, err := network[i].getLastBlock()
			assert.NoError(err)
			chainTip := blk.Header.Height

			logrus.WithField("node", i).
				WithField("height", chainTip).
				WithField("hash", hex.EncodeToString(blk.Header.Hash)).
				Info("local chain tip")

			if h <= 2 {
				h = chainTip
			} else {
				// Ensure not more than 2 rounds difference
				cond := (h-2 < chainTip && chainTip <= h+2)
				assert.True(cond, "a node falling behind error")
			}

			// Ensure consensus can reach N rounds
			if chainTip > 10 {
				return
			}
		}
	}
}
