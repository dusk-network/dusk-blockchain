// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package candidate

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config/genesis"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	assert "github.com/stretchr/testify/require"
)

func TestCandidateQueue(t *testing.T) {
	bus := eventbus.New()
	assert := assert.New(t)

	req := NewRequestor(bus)

	// Getting a block when no request is made should not result in it being
	// pushed to the candidateQueue
	_, err := req.ProcessCandidate("", message.New(topics.Candidate, *genesis.Decode()))
	assert.NoError(err)

	assert.Empty(req.candidateQueue)

	// Getting a block when requesting should make it end up in the queue
	c := genesis.Decode()

	req.setRequesting(true)

	_, err = req.ProcessCandidate("", message.New(topics.Candidate, *c))
	assert.NoError(err)

	assert.True(len(req.candidateQueue) == 1)
	c2 := <-req.candidateQueue
	assert.True(c.Equals(&c2))
}

func TestRequestor(t *testing.T) {
	bus := eventbus.New()
	assert := assert.New(t)

	req := NewRequestor(bus)

	c := genesis.Decode()

	streamer := eventbus.NewGossipStreamer(protocol.TestNet)

	l := eventbus.NewStreamListener(streamer)
	bus.Subscribe(topics.Gossip, l)

	cChan := make(chan block.Block, 1)

	go func() {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(2*time.Second))
		defer cancel()

		cm, err := req.RequestCandidate(ctx, c.Header.Hash)
		assert.NoError(err)

		cChan <- cm
	}()

	// Check if we receive a `GetCandidate` message
	m, err := streamer.Read()
	assert.NoError(err)

	assert.True(streamer.SeenTopics()[0] == topics.GetCandidate)
	assert.True(bytes.Equal(c.Header.Hash, m))

	_, err = req.ProcessCandidate("", message.New(topics.Candidate, *c))
	assert.NoError(err)

	// Wait for the candidate to be processed
	c2 := <-cChan
	assert.NotEmpty(t, c2)
	assert.True(c.Equals(&c2))
}
