// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package chain

import "github.com/dusk-network/dusk-blockchain/pkg/core/data/block"

// Ledger is the Chain interface used in tests.
type Ledger interface {
	TryNextConsecutiveBlockInSync(blk block.Block, kadcastHeight byte) error
	TryNextConsecutiveBlockOutSync(blk block.Block, kadcastHeight byte) error
	TryNextConsecutiveBlockIsValid(blk block.Block) error

	// StartConsensus starts the consensus loop that deals with start-and-stop
	// and result-fetch of the Consensus Spin.
	// The Consensus Spin is the loop that performs the Segregated Byzantine
	// Agreement over a single round.
	StartConsensus() error
	// StopConsensus signals the consensus loop to terminate if exists.
	StopConsensus()

	ProcessSyncTimerExpired(strPeerAddr string) error
}
