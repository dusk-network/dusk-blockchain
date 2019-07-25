package chainsync_test

import (
	"bytes"
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing/chainsync"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
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
	var topicBytes [15]byte
	copy(topicBytes[:], msg.Bytes()[0:15])
	topic := topics.ByteArrayToTopic(topicBytes)
	if topic != topics.GetBlocks {
		t.Fatal("did not receive expected GetBlocks message")
	}
}

// Check the behaviour of the ChainSynchronizer when receiving a block, when we
// are synced with other peers.
func TestSynchronizeSynced(t *testing.T) {
	cs, eb, _ := setupSynchronizer(t)

	// subscribe to topics.Block
	blockChan := make(chan *bytes.Buffer, 1)
	_ = eb.Subscribe(string(topics.Block), blockChan)

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
	if err := blk.Encode(buf); err != nil {
		panic(err)
	}

	return buf
}

func setupSynchronizer(t *testing.T) (*chainsync.ChainSynchronizer, *wire.EventBus, chan *bytes.Buffer) {
	eb := wire.NewEventBus()
	rpcBus := wire.NewRPCBus()
	responseChan := make(chan *bytes.Buffer, 100)
	counter := chainsync.NewCounter(eb)
	cs := chainsync.NewChainSynchronizer(eb, rpcBus, responseChan, counter)
	go respond(t, rpcBus)
	return cs, eb, responseChan
}

// Dummy goroutine which simply sends a random block back when the ChainSynchronizer
// requests the last block.
func respond(t *testing.T, rpcBus *wire.RPCBus) {
	r := <-wire.GetLastBlockChan
	r.RespChan <- *randomBlockBuffer(t, 0, 1)
}
