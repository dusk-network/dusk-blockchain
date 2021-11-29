// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package consensus

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/stretchr/testify/assert"
)

func TestQueueMaxCap(t *testing.T) {
	q := NewQueue()

	// Fill 10 rounds with a message
	for i := 0; i < 10; i++ {
		k := key.NewRandKeys()
		r := message.MockReduction(make([]byte, 32), uint64(i), 2, []key.Keys{k})
		q.PutEvent(uint64(i), 2, message.New(topics.Reduction, r))
	}

	assert.Equal(t, 10, q.items)

	// Clearing the queue to round 4 should set items to 5
	q.Clear(4)
	assert.Equal(t, 5, q.items)

	// Clearing the queue to round 10 should set items to 0
	q.Clear(10)
	assert.Equal(t, 0, q.items)

	k := key.NewRandKeys()
	r := message.MockReduction(make([]byte, 32), 0, 2, []key.Keys{k})

	// Fill queue with >4096 messages
	for i := 0; i < 4100; i++ {
		q.PutEvent(0, 2, message.New(topics.Reduction, r))
	}

	assert.Equal(t, 4096, q.items)

	// Flushing should empty queue
	q.Flush(0)
	assert.Equal(t, 0, q.items)

	for i := 9; i >= 0; i-- {
		k := key.NewRandKeys()
		r := message.MockReduction(make([]byte, 32), 5, uint8(i), []key.Keys{k})
		q.PutEvent(5, uint8(i), message.New(topics.Reduction, r))
		q.PutEvent(5, uint8(i), message.New(topics.AggrAgreement, r))
	}

	assert.Equal(t, 20, q.items)
	// Getting events from a step should remove 1 from items counter
	_ = q.GetEvents(5, 9)
	assert.Equal(t, 18, q.items)

	messages := q.Flush(5)
	last_step := uint8(0)

	for i := 0; i < 9; i++ {
		assert.Equal(t, messages[i].Category(), topics.AggrAgreement)
		hdr := messages[i].Payload().(message.Reduction).State()
		assert.GreaterOrEqual(t, hdr.Step, last_step)
		assert.Equal(t, uint8(i), hdr.Step)
		println(hdr.String())
		last_step = hdr.Step
	}

	last_step = uint8(0)

	for i := 9; i < 18; i++ {
		assert.Equal(t, messages[i].Category(), topics.Reduction)
		hdr := messages[i].Payload().(message.Reduction).State()
		assert.Equal(t, uint8(i-9), hdr.Step)
		assert.GreaterOrEqual(t, hdr.Step, last_step)
		last_step = hdr.Step
	}
}
