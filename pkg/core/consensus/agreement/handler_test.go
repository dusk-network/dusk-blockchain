// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package agreement

import (
	"bytes"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/sirupsen/logrus"
	assert "github.com/stretchr/testify/require"
)

// TestMockAgreementEvent tests the general layout of a mock Agreement (i.e. the BitSet).
func TestMockAgreementEvent(t *testing.T) {
	p, keys := consensus.MockProvisioners(50)
	hash, _ := crypto.RandEntropy(32)
	ev := message.MockAgreement(hash, 1, 3, keys, p)
	assert.NotEqual(t, 0, ev.VotesPerStep[0].BitSet)
	assert.NotEqual(t, 0, ev.VotesPerStep[1].BitSet)
}

func TestVoteVerification(t *testing.T) {
	// mocking voters
	p, keys := consensus.MockProvisioners(3)
	hash, _ := crypto.RandEntropy(32)
	ev := message.MockAgreement(hash, 1, 3, keys, p)
	handler := NewHandler(keys[0], *p, []byte{0, 0, 0, 0})
	assert.NoError(t, handler.Verify(ev), "problems in verification logic")
}

func TestGetVoterKeys(t *testing.T) {
	p, keys := consensus.MockProvisioners(3)
	hash, _ := crypto.RandEntropy(32)
	ev := message.MockAgreement(hash, 1, 3, keys, p)
	handler := NewHandler(keys[0], *p, []byte{0, 0, 0, 0})

	voterKeys, err := handler.getVoterKeys(ev)
	assert.Nil(t, err)

	// Ensure voterKeys only contains keys from `keys`
	for _, key := range voterKeys {
		found := false

		for _, k := range keys {
			if bytes.Equal(k.BLSPubKey, key) {
				found = true
			}
		}

		assert.True(t, found)
	}
}

func BenchmarkAgreementVerification(b *testing.B) {
	logrus.SetLevel(logrus.InfoLevel)

	const (
		workersCount      = 1
		provisionersCount = 64
		msgCount          = 50

		round = uint64(999)
		step  = uint8(3)
	)

	b.StopTimer()

	// mocking voters
	p, keys := consensus.MockProvisioners(provisionersCount)
	hash, _ := crypto.RandEntropy(32)
	ev := message.MockAgreement(hash, round, step, keys, p)
	handler := NewHandler(keys[0], *p, []byte{0, 0, 0})

	// Pre-generate committees for this round first and second reduction steps so
	// that handler.lock overhead is excluded
	_ = handler.Committee(round, step-1)
	_ = handler.Committee(round, step)

	a := &Accumulator{
		handler:            handler,
		verificationChan:   make(chan message.Agreement, 100),
		eventChan:          make(chan message.Agreement, 100),
		CollectedVotesChan: make(chan []message.Agreement, 1),
		storeMap:           newStoreMap(),
		workersQuitChan:    make(chan struct{}),
	}

	a.CreateWorkers(workersCount)

	b.StartTimer()

	for i := 0; i < msgCount; i++ {
		a.Process(ev)
	}

	// Drain all events to ensure all messages are processed
	c := msgCount
	for range a.eventChan {
		c--
		if c == 0 {
			return
		}
	}

	b.StopTimer()

	a.Stop()
	time.Sleep(100 * time.Millisecond)
}
