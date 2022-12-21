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
	NodeVersion = "0.6.1"

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

	// EPOCH used for stake operations.
	EPOCH = 2160
)
