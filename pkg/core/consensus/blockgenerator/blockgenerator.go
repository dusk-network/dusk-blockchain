// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package blockgenerator

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/blockgenerator/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
)

type candidateGenerator = candidate.Generator

type blockGenerator struct {
	candidateGenerator
}

// BlockGenerator ...
type BlockGenerator interface {
	candidateGenerator
}

// New creates a new BlockGenerator.
func New(e *consensus.Emitter, genPubKey *keys.PublicKey) BlockGenerator {
	return &blockGenerator{
		candidateGenerator: candidate.New(e, genPubKey),
	}
}

// Mock the block generator. If inert is true, no block will be generated (this
// simulates the score not reaching the threshold).
func Mock(e *consensus.Emitter, inert bool) BlockGenerator {
	return &blockGenerator{
		candidateGenerator: candidate.Mock(e),
	}
}
