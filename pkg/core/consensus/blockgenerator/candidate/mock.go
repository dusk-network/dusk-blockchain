// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package candidate

import (
	"context"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

// MockCandidate ...
func (m *mock) MockCandidate(sev message.ScoreProposal, previousBlock []byte) block.Block {
	if previousBlock == nil {
		previousBlock, _ = crypto.RandEntropy(32)
	}

	hdr := sev.State()

	b, err := m.GenerateBlock(hdr.Round, sev.Seed, sev.Proof, sev.Score, previousBlock, [][]byte{hdr.PubKeyBLS})
	if err != nil {
		panic(err)
	}

	return *b
}

type mock struct {
	*generator
}

func (m *mock) GenerateCandidateMessage(ctx context.Context, sev message.ScoreProposal, r consensus.RoundUpdate, step uint8) (*message.Score, error) {
	mockScore := message.MockScore(sev.State(), m.MockCandidate(sev, nil))
	return &mockScore, nil
}

// Mock the candidate generator.
func Mock(e *consensus.Emitter) Generator {
	key := keys.NewPublicKey()
	return &mock{
		generator: New(e, key).(*generator),
	}
}
