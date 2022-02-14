// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package genesis

import (
	"encoding/hex"
	"os"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
)

// DEFAULT_STATE_ROOT is the state root result of "rusk make state".
const DEFAULT_STATE_ROOT string = "d1e06f5fbfe56890dc55e972444d7ac3465af5648a843fd9e7043ede2711259e"

// Generate a genesis block. The constitution of the block depends on the passed
// config.
func Generate(c Config) *block.Block {
	// TODO: Populate this with real txs data from Rusk Transfer and Stake Contract
	if c.Transactions == nil {
		c.Transactions = make([]transactions.ContractCall, 0)
		c.Transactions = append(c.Transactions, transactions.MockTx())
	}

	var state_root []byte
	var err error

	state_root_str := DEFAULT_STATE_ROOT

	if state_root_override := os.Getenv("RUSK_STATE_ROOT"); len(state_root_override) > 0 {
		state_root_str = state_root_override
	}

	if state_root, err = hex.DecodeString(state_root_str); err != nil {
		panic(err)
	}

	h := &block.Header{
		Version:       0,
		Timestamp:     c.timestamp,
		Height:        0,
		PrevBlockHash: state_root,
		TxRoot:        nil,
		Seed:          c.seed,
		Certificate:   block.EmptyCertificate(),
		StateHash:     state_root,
	}

	b := &block.Block{
		Header: h,
		Txs:    c.Transactions,
	}

	// Set root and hash, since they have changed because of the adding of txs.
	root, err := b.CalculateRoot()
	if err != nil {
		panic(err)
	}

	b.Header.TxRoot = root

	hash, err := b.CalculateHash()
	if err != nil {
		panic(err)
	}

	b.Header.Hash = hash
	return b
}

// Decode marshals a genesis block into a buffer.
func Decode() *block.Block {
	cfg, err := GetPresetConfig(config.Get().General.Network)
	if err != nil {
		panic(err)
	}

	return Generate(cfg)
}
