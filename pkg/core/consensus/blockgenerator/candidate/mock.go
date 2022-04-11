// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package candidate

import (
	"context"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

// MockCandidate ...
func (m *mock) MockCandidate(hdr header.Header, previousBlock []byte) block.Block {
	if previousBlock == nil {
		previousBlock, _ = crypto.RandEntropy(32)
	}

	seed, _ := crypto.RandEntropy(32)

	b, err := m.GenerateBlock(hdr.Round, seed, previousBlock, 0, [][]byte{hdr.PubKeyBLS})
	if err != nil {
		panic(err)
	}

	return *b
}

type mock struct {
	*generator
}

func (m *mock) GenerateCandidateMessage(ctx context.Context, r consensus.RoundUpdate, step uint8) (*message.NewBlock, error) {
	hdr := header.Header{
		PubKeyBLS: make([]byte, 96),
		Round:     r.Round,
		Step:      step,
		BlockHash: make([]byte, 32),
	}

	cand := m.MockCandidate(hdr, nil)
	mockNewBlock := message.MockNewBlock(hdr, cand)
	return &mockNewBlock, nil
}

// Mock the candidate generator.
func Mock(e *consensus.Emitter) Generator {
	fn := func(ctx context.Context, txs []transactions.ContractCall, bh uint64, generator []byte) ([]transactions.ContractCall, []byte, error) {
		if len(txs) == 0 {
			// Function simulates Rusk ExecuteStateTransition.
			// That said, it should always provide a Distribute transaction
			txs = make([]transactions.ContractCall, 0)
			txs = append(txs, transactions.MockDistributeTx())
		}

		return txs, make([]byte, 32), nil
	}

	return &mock{
		generator: New(e, fn).(*generator),
	}
}
