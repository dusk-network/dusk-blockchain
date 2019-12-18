package candidate

import (
	"bytes"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-wallet/block"
	"github.com/stretchr/testify/assert"
)

// Ensure that the behaviour of the validator works as intended.
// It should republish blocks with a correct hash and root.
func TestValidatorValidBlock(t *testing.T) {
	eb := eventbus.New()
	streamer := eventbus.NewGossipStreamer(protocol.TestNet)
	eb.Subscribe(topics.Gossip, eventbus.NewStreamListener(streamer))
	eb.Register(topics.Gossip, processing.NewGossip(protocol.TestNet))
	v := newValidator(eb)

	cm := mockCandidateMessage(t)
	buf := new(bytes.Buffer)
	if err := Encode(buf, cm); err != nil {
		t.Fatal(err)
	}

	// Send it over to the validator
	if err := v.Process(buf); err != nil {
		t.Fatal(err)
	}

	// Should get same the block back, republished
	blkBytes, err := streamer.Read()
	if err != nil {
		t.Fatal(err)
	}

	decodedBlk := block.NewBlock()
	if err := marshalling.UnmarshalBlock(bytes.NewBuffer(blkBytes), decodedBlk); err != nil {
		t.Fatal(err)
	}

	assert.True(t, cm.Block.Equals(decodedBlk))
}

// Ensure that blocks with an invalid hash or tx root will not be
// republished.
func TestValidatorInvalidBlock(t *testing.T) {
	eb := eventbus.New()
	gossipChan := make(chan bytes.Buffer, 1)
	eb.Subscribe(topics.Gossip, eventbus.NewChanListener(gossipChan))
	v := newValidator(eb)

	cm := mockCandidateMessage(t)
	buf := new(bytes.Buffer)
	if err := Encode(buf, cm); err != nil {
		t.Fatal(err)
	}

	// Remove one of the transactions to remove the integrity of
	// the merkle root
	cm.Block.Txs = cm.Block.Txs[1:]
	buf = new(bytes.Buffer)
	if err := Encode(buf, cm); err != nil {
		t.Fatal(err)
	}

	if err := v.Process(buf); err == nil {
		t.Fatal("processing a block with an invalid hash should return an error")
	}

	// Should not be anything republished
	select {
	case <-time.After(1 * time.Second):
		// Success
	case <-gossipChan:
		t.Fatal("not supposed to get anything on gossipChan")
	}
}
