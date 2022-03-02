// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package chain

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
)

// MockVerifier is a mock for the chain.Verifier interface.
type MockVerifier struct{}

// SanityCheckBlockchain on first N blocks and M last blocks.
func (v *MockVerifier) SanityCheckBlockchain(uint64, uint64) error {
	return nil
}

// SanityCheckBlock will verify whether a block is valid according to the rules of the consensus.
func (v *MockVerifier) SanityCheckBlock(prevBlock block.Block, blk block.Block) error {
	return nil
}

// MockLoader is the mock of the DB loader to help testing the chain.
type MockLoader struct {
	blockchain []block.Block
}

// NewMockLoader creates a Mockup of the Loader interface.
func NewMockLoader() Loader {
	mockchain := make([]block.Block, 0)
	return &MockLoader{mockchain}
}

// Height returns the height currently known by the Loader.
func (m *MockLoader) Height() (uint64, error) {
	return uint64(len(m.blockchain)), nil
}

// LoadTip of the chain.
func (m *MockLoader) LoadTip() (*block.Block, []byte, error) {
	return &m.blockchain[len(m.blockchain)], nil, nil
}

// SanityCheckBlockchain on first N blocks and M last blocks.
func (m *MockLoader) SanityCheckBlockchain(uint64, uint64, uint64) error {
	return nil
}

// Clear the mock.
func (m *MockLoader) Clear() error {
	return nil
}

// Close the mock.
func (m *MockLoader) Close(driver string) error {
	return nil
}

// Append the block to the internal blockchain representation.
func (m *MockLoader) Append(blk *block.Block) error {
	m.blockchain = append(m.blockchain, *blk)
	return nil
}

// BlockAt the block to the internal blockchain representation.
func (m *MockLoader) BlockAt(index uint64) (block.Block, error) {
	return m.blockchain[index], nil
}
