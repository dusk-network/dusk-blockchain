package chainsync_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	assert "github.com/stretchr/testify/require"
)

// Check the behavior of the ChainSynchronizer when receiving a block, when we
// are sufficiently behind the chain tip.
func TestSynchronizeBehind(t *testing.T) {
	assert := assert.New(t)
	cs, eb, responseChan := setupSynchronizer()
	// Create a listener for HighestSeen topic
	highestSeenChan := make(chan message.Message, 1)
	eb.Subscribe(topics.HighestSeen, eventbus.NewChanListener(highestSeenChan))

	// Create a block that is a few rounds in the future
	height := uint64(5)
	blk := randomBlockBuffer(height, 20)

	if err := cs.HandleBlock(blk, "test_peer_addr"); err != nil {
		t.Fatal(err)
	}

	msg := <-responseChan

	// Check topic
	topic, err := topics.Extract(msg)
	assert.NoError(err)
	if topic != topics.GetBlocks {
		t.Fatal("did not receive expected GetBlocks message")
	}

	// TODO: Check highest seen
	// highestSeenHeightMsg := <-highestSeenChan
	// highestSeenHeight, err := message.ConvU64(highestSeenHeightMsg.Payload())

	//  assert.NoError(err)
	//assert.Equal(highestSeenHeight, height)
}

// Check the behavior of the ChainSynchronizer when receiving a block, when we
// are synced with other peers.
func TestSynchronizeSynced(t *testing.T) {
	assert := assert.New(t)
	cs, eb, _ := setupSynchronizer()

	// subscribe to topics.Block
	blockChan := make(chan message.Message, 1)
	listener := eventbus.NewChanListener(blockChan)
	_ = eb.Subscribe(topics.Block, listener)

	// Make a block which should follow our genesis block
	blk := randomBlockBuffer(1, 20)

	assert.NoError(cs.HandleBlock(blk, "test_peer_addr"))
	// The synchronizer should put this block on the blockChan
	msg := <-blockChan

	// Payload should be of type block.Block
	assert.NotPanics(func() { _ = msg.Payload().(block.Block) })
}

// Returns an encoded representation of a `helper.RandomBlock`.
func randomBlockBuffer(height uint64, txBatchCount uint16) *bytes.Buffer {
	blk := helper.RandomBlock(height, txBatchCount)
	buf := new(bytes.Buffer)
	if err := message.MarshalBlock(buf, blk); err != nil {
		panic(err)
	}

	return buf
}

func setupSynchronizer() (*chainsync.ChainSynchronizer, *eventbus.EventBus, chan *bytes.Buffer) {
	eb := eventbus.New()
	rpcBus := rpcbus.New()
	responseChan := make(chan *bytes.Buffer, 100)
	counter := chainsync.NewCounter(eb)
	cs := chainsync.NewChainSynchronizer(eb, rpcBus, responseChan, counter)
	respond(rpcBus)
	return cs, eb, responseChan
}

// Dummy goroutine which simply sends a random block back when the ChainSynchronizer
// requests the last block.
func respond(rpcBus *rpcbus.RPCBus) {
	g := make(chan rpcbus.Request, 1)
	rpcBus.Register(topics.GetLastBlock, g)
	go func() {
		r := <-g
		r.RespChan <- rpcbus.NewResponse(*helper.RandomBlock(0, 1), nil)
	}()
}
