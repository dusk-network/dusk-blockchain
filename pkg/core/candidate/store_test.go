package candidate

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/block"
	"github.com/stretchr/testify/assert"
)

// Test basic storage-related functionality.
func TestStoreFetchClear(t *testing.T) {
	c := newStore()

	// Store a candidate
	candidate := mockCandidateMessage(t)
	assert.NoError(t, c.storeCandidateMessage(*candidate))

	// Fetch it now
	// Hash is genesis hash
	genesis := config.DecodeGenesis()
	fetched := c.fetchCandidateMessage(genesis.Header.Hash)
	assert.NotNil(t, fetched)

	// Correctness checks
	assert.True(t, genesis.Equals(fetched.Block))
	assert.True(t, fetched.Certificate.Equals(block.EmptyCertificate()))

	// Check that Clear empties the entire candidate store
	n := c.Clear(1)
	assert.Equal(t, 1, n)
	assert.Empty(t, c.messages)
}

// Test the candidate request functionality.
func TestRequestCandidate(t *testing.T) {
	eb := eventbus.New()
	rpc := rpcbus.New()
	b := newBroker(eb, rpc)
	go b.Listen()

	streamer := eventbus.NewGossipStreamer(protocol.TestNet)
	eb.Subscribe(topics.Gossip, eventbus.NewStreamListener(streamer))
	gossip := processing.NewGossip(protocol.TestNet)
	eb.Register(topics.Gossip, gossip)

	// Fetch a candidate we don't have
	doneChan := make(chan struct{}, 1)
	genesis := config.DecodeGenesis()
	go func() {
		blkBuf, err := rpc.Call(rpcbus.GetCandidate, rpcbus.Request{*bytes.NewBuffer(genesis.Header.Hash), make(chan rpcbus.Response, 1)}, 0)
		assert.NoError(t, err)

		blk := block.NewBlock()
		assert.NoError(t, marshalling.UnmarshalBlock(&blkBuf, blk))
		assert.True(t, blk.Equals(genesis))
		doneChan <- struct{}{}
	}()
	// Make sure we receive a GetCandidate message
	m, err := streamer.Read()
	if err != nil {
		t.Fatal(err)
	}

	// Make sure we got a request for the genesis block
	assert.True(t, bytes.Equal(m, genesis.Header.Hash))

	// Send genesis back as a Candidate message
	cm := mockCandidateMessage(t)
	buf := new(bytes.Buffer)
	if err := Encode(buf, cm); err != nil {
		t.Fatal(err)
	}

	eb.Publish(topics.Candidate, buf)
}

// Mocks a candidate message
func mockCandidateMessage(t *testing.T) *Candidate {
	genesis := config.DecodeGenesis()
	cert := block.EmptyCertificate()

	return &Candidate{genesis, cert}
}
