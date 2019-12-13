package chain

import (
	"bytes"
	"errors"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-wallet/block"
	"github.com/stretchr/testify/assert"
)

// Test basic storage-related functionality.
func TestStoreFetchClear(t *testing.T) {
	eb := eventbus.New()
	c := newCandidateStore(eb)

	// Store a candidate
	candidate := mockCandidateMessage(t)
	assert.NoError(t, c.storeCandidateMessage(candidate))

	// Fetch it now
	// Hash is genesis hash
	genesis := config.DecodeGenesis()
	fetched, err := c.fetchCandidateMessage(genesis.Header.Hash)
	assert.NoError(t, err)

	// Correctness checks
	assert.True(t, genesis.Equals(fetched.blk))
	assert.True(t, fetched.cert.Equals(block.EmptyCertificate()))

	// Check that Clear empties the entire candidate store
	n := c.Clear(1)
	assert.Equal(t, 1, n)
	assert.Empty(t, c.messages)
}

// Test the candidate request functionality.
func TestRequestCandidate(t *testing.T) {
	eb := eventbus.New()
	c := newCandidateStore(eb)
	streamer := eventbus.NewGossipStreamer(protocol.TestNet)
	eb.Subscribe(topics.Gossip, eventbus.NewStreamListener(streamer))
	gossip := processing.NewGossip(protocol.TestNet)
	eb.Register(topics.Gossip, gossip)

	// Fetch a candidate we don't have
	doneChan := make(chan struct{}, 1)
	genesis := config.DecodeGenesis()
	go func(genesis *block.Block, doneChan chan struct{}) {
		fetched, err := c.fetchCandidateMessage(genesis.Header.Hash)
		assert.NoError(t, err)
		assert.True(t, fetched.blk.Equals(genesis))
		doneChan <- struct{}{}
	}(genesis, doneChan)

	// Make sure we receive a GetCandidate message
	m, err := streamer.Read()
	if err != nil {
		t.Fatal(err)
	}

	// Make sure we got a request for the genesis block
	assert.True(t, bytes.Equal(m, genesis.Header.Hash))

	// Send genesis back as a Candidate message
	cm := mockCandidateMessage(t)
	eb.Publish(topics.Candidate, &cm)

	// Ensure the goroutine exits without failure
	<-doneChan
}

// Test that the candidate request functionality will time out
// after a while, so that it won't lock up the Chain.
func TestRequestCandidateTimeout(t *testing.T) {
	eb := eventbus.New()
	c := newCandidateStore(eb)

	// Fetch a candidate we don't have
	genesis := config.DecodeGenesis()
	_, err := c.fetchCandidateMessage(genesis.Header.Hash)
	// Should get a request timeout error eventually
	assert.Equal(t, errors.New("request timeout"), err)
}

// Mocks a candidate message
func mockCandidateMessage(t *testing.T) bytes.Buffer {
	genesis := config.DecodeGenesis()
	cert := block.EmptyCertificate()

	buf := new(bytes.Buffer)
	if err := marshalling.MarshalBlock(buf, genesis); err != nil {
		t.Fatal(err)
	}

	if err := marshalling.MarshalCertificate(buf, cert); err != nil {
		t.Fatal(err)
	}

	return *buf
}
