// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package score

import (
	"context"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

type mock struct {
	*consensus.Emitter
	inert bool
}

func (m *mock) Generate(ctx context.Context, r consensus.RoundUpdate, step uint8) message.ScoreProposal {
	hash, _ := crypto.RandEntropy(32)

	hdr := header.Header{
		Round:     r.Round,
		Step:      step,
		BlockHash: hash,
		PubKeyBLS: m.Keys.BLSPubKeyBytes,
	}

	if m.inert {
		return message.EmptyScoreProposal(hdr)
	}

	return message.MockScoreProposal(hdr)
}

// Mock a score generator
func Mock(e *consensus.Emitter, inert bool) Generator {
	return &mock{e, inert}
}
