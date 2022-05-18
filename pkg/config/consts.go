// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package config

import "time"

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

	MaxBlockTime = 360 // maximum block time in seconds

	// KadcastInitialHeight sets the default initial height for Kadcast broadcast algorithm.
	KadcastInitialHeight byte = 128 + 1

	// The dusk-blockchain executable version.
	NodeVersion = "0.5.0"

	// The shared API version (between dusk-blockchain and rusk).
	InteropVersion = "0.1.0"

	// TESTNET_GENESIS_HASH is the default genesis hash for testnet.
	TESTNET_GENESIS_HASH = "bef3d7b99c3e4dafa6bebee5be63e78d72abb8ec2ae98e1db3a0618f7f2b4977"

	// DEFAULT_STATE_ROOT is the state root result of "rusk make state".
	DEFAULT_STATE_ROOT string = "f100e6fbc3297b6f2f084c0e1fe3833e0ed050830e790e8efd68536b75baa919"

	// Consensus-related settings
	// Protocol-based consensus step time.
	ConsensusTimeOut = 5 * time.Second

	// ConsensusTimeThreshold consensus time in seconds above which we don't throttle it.
	ConsensusTimeThreshold = 10

	// ConsensusQuorumThreshold is consensus quorum percentage.
	ConsensusQuorumThreshold = 0.67

	// ConsensusMaxStep consensus max step number.
	ConsensusMaxStep = uint8(213)

	// ConsensusMaxCommitteeSize represents the maximum size of the committee in
	// 1st_Reduction, 2th_Reduction and Agreement phases.
	ConsensusMaxCommitteeSize = 64

	// ConsensusMaxCommitteeSize represents the maximum size of the committee in
	// Selection phase.
	ConsensusSelectionMaxCommitteeSize = 1
)

// KadcastInitHeader is used as default initial kadcast message header.
var KadcastInitHeader = []byte{KadcastInitialHeight}
