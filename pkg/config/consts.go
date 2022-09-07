// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package config

// A single point of constants definition.
const (
	// DUSK is one whole unit of DUSK.
	DUSK = uint64(1_000_000_000)

	// Default Block Gas limit.
	// This value should handle up to ~20 transfers (or ~5 InterContract calls) per block.
	DefaultBlockGasLimit = 5 * DUSK

	// MaxTxSetSize defines the maximum amount of transactions.
	// It is TBD along with block size and processing.MaxFrameSize.
	MaxTxSetSize = 825000

	// Maximum number of blocks to be requested/delivered on a single syncing session with a peer.
	MaxInvBlocks = 500

	MaxBlockTime = 360 // maximum block time in seconds

	// KadcastInitialHeight sets the default initial height for Kadcast broadcast algorithm.
	KadcastInitialHeight byte = 128

	// The dusk-blockchain executable version.
	NodeVersion = "0.6.0"

	// TESTNET_GENESIS_HASH is the default genesis hash for testnet.
	TESTNET_GENESIS_HASH = "1c0125633b32cec077a6723ca943435b0effb89343c439c70793ff1d6f8066db"

	// TESTNET_STATE_ROOT is the state root result of
	// "rusk `RUSK_PREBUILT_CONTRACTS=true RUSK_BUILD_TESTNET=true make state`".
	TESTNET_STATE_ROOT string = "ff3e64a0ce8332396a43827094b9c3de7a29741fd81edc9b493f4e236d3228fb"

	// HARNESS_GENESIS_HASH is the default genesis hash for harness (local net).
	HARNESS_GENESIS_HASH = "cf1bec522ee62aa832ef2cad62b2b73dda5ea6c95110add9a074f2600517bf42"

	// HARNESS_STATE_ROOT is the state root result of
	// "rusk `RUSK_PREBUILT_CONTRACTS=true RUSK_BUILD_TESTNET=false make state`".
	HARNESS_STATE_ROOT string = "bf07fa6a5e43168c1cba95b17d0d414f6dc050b2d6056905f6049e66bb4ed307"

	// Consensus-related settings
	// Protocol-based consensus step time.
	DefaultConsensusTimeOutSeconds = 5

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

	// RuskVersion is the version of the supported rusk binary.
	RuskVersion = "0.5.0"

	// GetCandidateReceivers is a redundancy factor on retrieving a missing candidate block.
	GetCandidateReceivers = 7
)
