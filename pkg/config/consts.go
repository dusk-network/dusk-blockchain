// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package config

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config/genesis"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
)

// A single point of constants definition.
const (
	// GeneratorReward is the amount of Block generator default reward.
	// TODO: TBD.
	GeneratorReward = 50 * wallet.DUSK

	MinFee = uint64(100)

	// MaxLockTime is the maximum amount of time a consensus transaction (stake, bid)
	// can be locked up for.
	MaxLockTime = uint64(250000)

	// Maximum number of blocks to be requested/delivered on a single syncing session with a peer.
	MaxInvBlocks = 500

	// Protocol-based consensus step time.
	ConsensusTimeOut = 5 * time.Second

	// KadcastInitialHeight sets the default initial height for Kadcast broadcast algorithm.
	KadcastInitialHeight byte = 128
)

var (
	KadcastInitHeader = []byte{KadcastInitialHeight}
)

// DecodeGenesis marshals a genesis block into a buffer.
func DecodeGenesis() *block.Block {
	cfg, err := genesis.GetPresetConfig(Get().General.Network)
	if err != nil {
		panic(err)
	}

	return genesis.Generate(cfg)
}
