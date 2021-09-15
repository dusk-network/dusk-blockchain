// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package agreement

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	crypto "github.com/dusk-network/dusk-crypto/hash"
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
	handler := NewHandler(keys[0], *p)
	assert.NoError(t, handler.Verify(ev), "problems in verification logic")
}

func TestGetVoterKeys(t *testing.T) {
	p, keys := consensus.MockProvisioners(3)
	hash, _ := crypto.RandEntropy(32)
	ev := message.MockAgreement(hash, 1, 3, keys, p)
	handler := NewHandler(keys[0], *p)

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
