package testing

import (
	stdtesting "testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestConsensus performs a integration testing upon complete consensus logic
// TestConsensus passing means the consensus phases are properly assembled
func TestConsensus(t *stdtesting.T) {

	t.SkipNow()

	assert := assert.New(t)

	// Create Gossip Router
	streamer := eventbus.NewRouterStreamer()
	streamListener := eventbus.NewStreamListener(streamer)

	network := make([]mockNode, 0)
	networkSize := 3

	// Initialize consensus participants
	for i := 0; i < networkSize; i++ {
		_, pk := transactions.MockKeys()
		node := newMockNode(pk, streamListener, assert)

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
		for i := 0; i < len(network); i++ {

			// Trace chain tip of all nodes
			blk, err := network[i].getLastBlock()
			assert.NoError(err)

			logrus.WithField("node", i).
				WithField("height", blk.Header.Height).
				Info("local chainTip")

			// Main check point to ensure test passes
			if blk.Header.Height > 10 {
				return
			}
		}
	}
}
