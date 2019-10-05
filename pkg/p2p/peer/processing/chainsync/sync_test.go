package chainsync_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/stretchr/testify/assert"
)

// Check the behaviour of the ChainSynchronizer when receiving a block, when we
// are sufficiently behind the chain tip.
func TestSynchronizeBehind(t *testing.T) {
	cs, _, responseChan := setupSynchronizer(t)

	// Create a block that is a few rounds in the future
	blk := randomBlockBuffer(t, 5, 20)

	if err := cs.Synchronize(blk, "test_peer"); err != nil {
		t.Fatal(err)
	}

	msg := <-responseChan

	// Check topic
	topic, err := topics.Extract(msg)
	assert.NoError(t, err)
	if topic != topics.GetBlocks {
		t.Fatal("did not receive expected GetBlocks message")
	}
}

// Check the behaviour of the ChainSynchronizer when receiving a block, when we
// are synced with other peers.
func TestSynchronizeSynced(t *testing.T) {
	cs, eb, _ := setupSynchronizer(t)

	// subscribe to topics.Block
	blockChan := make(chan bytes.Buffer, 1)
	listener := eventbus.NewChanListener(blockChan)
	_ = eb.Subscribe(string(topics.Block), listener)

	// Make a block which should follow our genesis block
	blk := randomBlockBuffer(t, 1, 20)

	if err := cs.Synchronize(blk, "test_peer"); err != nil {
		t.Fatal(err)
	}

	// The synchronizer should put this block on the blockChan
	<-blockChan
}

// Returns an encoded representation of a `helper.RandomBlock`.
func randomBlockBuffer(t *testing.T, height uint64, txBatchCount uint16) *bytes.Buffer {
	blk := helper.RandomBlock(t, height, txBatchCount)
	buf := new(bytes.Buffer)
	if err := block.Marshal(buf, blk); err != nil {
		panic(err)
	}

	return buf
}

func setupSynchronizer(t *testing.T) (*chainsync.ChainSynchronizer, *eventbus.EventBus, chan *bytes.Buffer) {
	eb := eventbus.New()
	rpcBus := rpcbus.New()
	responseChan := make(chan *bytes.Buffer, 100)
	counter := chainsync.NewCounter(eb)
	cs := chainsync.NewChainSynchronizer(eb, rpcBus, responseChan, counter)
	go respond(t, rpcBus)
	return cs, eb, responseChan
}

// Dummy goroutine which simply sends a random block back when the ChainSynchronizer
// requests the last block.
func respond(t *testing.T, rpcBus *rpcbus.RPCBus) {
	r := <-rpcbus.GetLastBlockChan
	r.RespChan <- *randomBlockBuffer(t, 0, 1)
}
