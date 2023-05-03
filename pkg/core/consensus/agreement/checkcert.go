// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package agreement

import (
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
)

// CheckBlockCertificate ensures that the block certificate is valid.
func CheckBlockCertificate(provisioners user.Provisioners, blk block.Block, seed []byte) error {
	// TODO: this should be set back to 1, once we fix this issue:
	// https://github.com/dusk-network/dusk-blockchain/issues/925
	if blk.Header.Height < 2 {
		return nil
	}

	// First, lets get the actual reduction steps
	// These would be the two steps preceding the one on the certificate
	stepOne := (blk.Header.Iteration-1)*3 + 2
	stepTwo := (blk.Header.Iteration-1)*3 + 3

	stepOneBatchedSig := blk.Header.Certificate.StepOneBatchedSig
	stepTwoBatchedSig := blk.Header.Certificate.StepTwoBatchedSig

	// Now, check the certificate's correctness for both reduction steps
	if err := checkBlockCertificateForStep(stepOneBatchedSig, blk.Header.Certificate.StepOneCommittee, blk.Header.Height, stepOne, provisioners, blk.Header.Hash, seed); err != nil {
		return err
	}

	return checkBlockCertificateForStep(stepTwoBatchedSig, blk.Header.Certificate.StepTwoCommittee, blk.Header.Height, stepTwo, provisioners, blk.Header.Hash, seed)
}

func checkBlockCertificateForStep(batchedSig []byte, bitSet uint64, round uint64, step uint8, provisioners user.Provisioners, blockHash, seed []byte) error {
	size := config.ConsensusMaxCommitteeSize
	committee := provisioners.CreateVotingCommittee(seed, round, step, size)
	subcommittee := committee.IntersectCluster(bitSet)

	stepVoters := subcommittee.TotalOccurrences()
	quorumTarget := quorum(size)

	if stepVoters < quorumTarget {
		return fmt.Errorf("vote set too small - %v/%v", stepVoters, quorumTarget)
	}

	apk, err := AggregatePks(&provisioners, subcommittee.Set)
	if err != nil {
		return err
	}

	return header.VerifySignatures(round, step, blockHash, apk, batchedSig)
}

func logStepVotes(msgStep uint8, msgRound uint64, msgHash []byte, h *handler, votesPerStep []*message.StepVotes, ru consensus.RoundUpdate) {
	quorumTarget := h.Quorum(msgRound)

	for i, votes := range votesPerStep {
		// the beginning step is the same of the second reduction. Since the
		// consensus steps start at 1, this is always a multiple of 3
		// The first reduction step is one less
		step := msgStep - 1 + uint8(i)

		// Committee the sortition determines for this round
		committee := h.Committee(msgRound, step)

		// subcommittee is a subset of the committee members that voted - the so-called quorum-committee
		subcommittee := committee.IntersectCluster(votes.BitSet)
		stepVoters := subcommittee.TotalOccurrences()

		log := consensus.WithFields(msgRound, step, "verification failed_step_votes",
			msgHash, h.BLSPubKey, &committee, &subcommittee, &h.Provisioners)

		log.WithField("bitset", votes.BitSet).
			// number of committee members for current (round, step, seed) tuple
			WithField("comm", committee.Len()).
			// number of subcommittee members
			WithField("subcomm", subcommittee.Len()).
			WithField("t_votes", stepVoters).
			WithField("target_votes", quorumTarget).
			WithField("prev_seed", util.StringifyBytes(ru.Seed)).
			WithField("prev_hash", util.StringifyBytes(ru.Hash)).
			Info()
	}
}
