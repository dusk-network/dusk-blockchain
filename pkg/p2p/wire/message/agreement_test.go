// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package message_test

import (
	"reflect"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestCopy(t *testing.T) {
	assert := assert.New(t)
	aggro := RandAgreement()
	cpy := aggro.Copy().(message.Agreement)
	red := aggro.State()
	h := cpy.State()

	assert.True(reflect.DeepEqual(aggro, cpy))

	// HEADER
	assert.Equal(red.Round, h.Round)
	assert.Equal(red.Step, h.Step)
	assert.Equal(red.BlockHash, h.BlockHash)
	assert.Equal(red.PubKeyBLS, h.PubKeyBLS)

	// SignedVotes
	assert.Equal(aggro.Signature(), cpy.Signature())

	// VotesPerStep
	for i, vps := range aggro.VotesPerStep {
		assert.Equal(vps, cpy.VotesPerStep[i])
	}
}

func RandAgreement() message.Agreement {
	p, keys := consensus.MockProvisioners(50)
	hash, _ := crypto.RandEntropy(32)
	return message.MockAgreement(hash, 44, 6, keys, p)
}
