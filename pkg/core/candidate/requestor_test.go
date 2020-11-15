package candidate

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	assert "github.com/stretchr/testify/require"
)

func TestCandidateQueue(t *testing.T) {
	bus, rpcBus := eventbus.New(), rpcbus.New()
	assert := assert.New(t)

	req := NewRequestor(bus, rpcBus)
	go req.Listen(context.Background())

	// Getting a block when no request is made should not result in it being
	// pushed to the candidateQueue
	_, err := req.ProcessCandidate(message.New(topics.Candidate, mockCandidate()))
	assert.NoError(err)

	assert.Empty(req.candidateQueue)

	// Getting a desired block should make it end up in the queue
	c := mockCandidate()
	req.setRequestHash(c.Block.Header.Hash)
	_, err = req.ProcessCandidate(message.New(topics.Candidate, c))
	assert.NoError(err)

	assert.True(len(req.candidateQueue) == 1)
	c2 := <-req.candidateQueue
	assert.True(c.Block.Equals(c2.Block))
}

func TestRequestor(t *testing.T) {
	bus, rpcBus := eventbus.New(), rpcbus.New()
	assert := assert.New(t)

	req := NewRequestor(bus, rpcBus)
	go req.Listen(context.Background())

	c := mockCandidate()

	streamer := eventbus.NewGossipStreamer(protocol.TestNet)
	bus.Subscribe(topics.Gossip, eventbus.NewStreamListener(streamer))
	cChan := make(chan message.Candidate, 1)

	go func() {
		cm, err := rpcBus.Call(topics.GetCandidate, rpcbus.NewRequest(c.Block.Header.Hash), 2*time.Second)
		assert.NoError(err)
		cChan <- cm.(message.Candidate)
	}()

	// Check if we receive a `GetCandidate` message
	m, err := streamer.Read()
	assert.NoError(err)

	assert.True(streamer.SeenTopics()[0] == topics.GetCandidate)
	assert.True(bytes.Equal(c.Block.Header.Hash, m))

	_, err = req.ProcessCandidate(message.New(topics.Candidate, c))
	assert.NoError(err)

	// Wait for the candidate to be processed
	c2 := <-cChan
	assert.NotEmpty(t, c2)
	assert.True(c.Block.Equals(c2.Block))
}

// Mocks a candidate message. It is not in the message package since it uses
// the genesis block as mockup block
//nolint:unused
func mockCandidate() message.Candidate {
	genesis := config.DecodeGenesis()
	cert := block.EmptyCertificate()
	return message.MakeCandidate(genesis, cert)
}
