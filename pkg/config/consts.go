// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package config

import (
	"time"
)

// A single point of constants definition.
const (
	// DUSK is one whole unit of DUSK.
	DUSK = uint64(1_000_000_000)

	// Default Block Gas limit.
	BlockGasLimit = 1000 * DUSK

	// MaxTxSetSize defines the maximum amount of transactions.
	// It is TBD along with block size and processing.MaxFrameSize.
	MaxTxSetSize = 825000

	// Maximum number of blocks to be requested/delivered on a single syncing session with a peer.
	MaxInvBlocks = 500

	// Protocol-based consensus step time.
	ConsensusTimeOut = 5 * time.Second

	// ConsensusTimeThreshold consensus time in seconds above which we don't throttle it.
	ConsensusTimeThreshold = 10

	MaxBlockTime = 360 // maximum block time in seconds

	// KadcastInitialHeight sets the default initial height for Kadcast broadcast algorithm.
	KadcastInitialHeight byte = 128

	// The dusk-blockchain executable version.
	NodeVersion = "0.5.0"

	// The shared API version (between dusk-blockchain and rusk).
	InteropVersion = "0.1.0"

	// TESTNET_GENESIS_HASH is the default genesis hash for testnet.
	TESTNET_GENESIS_HASH = "3f2f22608ff4657c488c029eb0fcf372bcc419531c2e578f3e5f957c39b171fb"

	// DEFAULT_STATE_ROOT is the state root result of "rusk make state".
	DEFAULT_STATE_ROOT string = "b40ba0dac43dd7bfa8e390c05e9da481a9598bed0689ef81c6eb52e86de6deac"
)

// KadcastInitHeader is used as default initial kadcast message header.
var KadcastInitHeader = []byte{KadcastInitialHeight}
