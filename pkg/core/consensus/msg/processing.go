package msg

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/sortition"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// Process is a top-level message processing function for the consensus.
func Process(ctx *user.Context, msg *payload.MsgConsensus) *prerror.PrError {
	// Verify Ed25519 signature
	edMsg := new(bytes.Buffer)
	if err := msg.EncodeSignable(edMsg); err != nil {
		return prerror.New(prerror.High, err)
	}

	if !ctx.EDVerify(msg.PubKey, edMsg.Bytes(), msg.Signature) {
		return prerror.New(prerror.Low, errors.New("ed25519 verification failed"))
	}

	// Check if this person has staked
	pkEd := hex.EncodeToString(msg.PubKey)
	if ctx.NodeWeights[pkEd] == 0 {
		return prerror.New(prerror.Low, errors.New("voter has not staked"))
	}

	// Version check
	if ctx.Version != msg.Version {
		return prerror.New(prerror.Low, errors.New("version mismatch"))
	}

	// Check if we're on the same round
	if ctx.Round != msg.Round {
		return prerror.New(prerror.Low, errors.New("round mismatch"))
	}

	// Check if we're on the same chain
	if !bytes.Equal(msg.PrevBlockHash, ctx.LastHeader.Hash) {
		return prerror.New(prerror.Low, errors.New("voter is on a different chain"))
	}

	// Proceed to more specific checks
	return verifyPayload(ctx, msg)
}

// Lower-level message processing function. This function determines the payload type,
// applies the proper verification functions, and then sends it off to the appropriate channel.
func verifyPayload(ctx *user.Context, msg *payload.MsgConsensus) *prerror.PrError {
	switch msg.Payload.Type() {
	case consensusmsg.CandidateScoreID:
		// TODO: add actual verification code for score messages
		ctx.CandidateScoreChan <- msg
		return nil
	case consensusmsg.CandidateID:
		// Block was already verified upon reception, so we don't do anything else here.
		ctx.CandidateChan <- msg
		return nil
	case consensusmsg.ReductionID:
		// Check if we're on the same step
		if ctx.Step != msg.Step {
			return prerror.New(prerror.Low, errors.New("step mismatch"))
		}

		// Check if this node is eligible to vote
		votes := sortition.Verify(ctx.CurrentCommittee, msg.PubKey)
		if votes == 0 {
			return prerror.New(prerror.Low, errors.New("node is not included in current committee"))
		}

		pl := msg.Payload.(*consensusmsg.Reduction)
		if !verifyBLSKey(ctx, msg.PubKey, pl.PubKeyBLS) {
			return prerror.New(prerror.Low, errors.New("BLS key mismatch"))
		}

		if err := verifyReduction(ctx, pl); err != nil {
			return err
		}

		ctx.ReductionChan <- msg
		return nil
	case consensusmsg.SetAgreementID:
		// Check what step this agreement message is from, and reconstruct the committee
		// for that step.
		committee, err := sortition.CreateCommittee(ctx.Round, ctx.W, msg.Step,
			uint8(len(ctx.CurrentCommittee)), ctx.Committee, ctx.NodeWeights)
		if err != nil {
			return prerror.New(prerror.High, err)
		}

		// Check if this node is eligible to vote
		votes := sortition.Verify(committee, msg.PubKey)
		if votes == 0 {
			return prerror.New(prerror.Low, errors.New("node is not included in committee"))
		}

		pl := msg.Payload.(*consensusmsg.SetAgreement)
		if err := verifyVoteSet(ctx, pl.VoteSet, pl.BlockHash, msg.Step); err != nil {
			return err
		}

		ctx.SetAgreementChan <- msg
		return nil
	case consensusmsg.SigSetCandidateID:
		pl := msg.Payload.(*consensusmsg.SigSetCandidate)
		if err := verifySigSetCandidate(ctx, pl, msg.Step); err != nil {
			return err
		}

		ctx.SigSetCandidateChan <- msg
		return nil
	case consensusmsg.SigSetVoteID:
		// Check if we're on the same step
		if ctx.Step != msg.Step {
			return prerror.New(prerror.Low, errors.New("step mismatch"))
		}

		// Check if this node is eligible to vote
		votes := sortition.Verify(ctx.CurrentCommittee, msg.PubKey)
		if votes == 0 {
			return prerror.New(prerror.Low, errors.New("node is not included in current committee"))
		}

		pl := msg.Payload.(*consensusmsg.SigSetVote)
		if !verifyBLSKey(ctx, msg.PubKey, pl.PubKeyBLS) {
			return prerror.New(prerror.Low, errors.New("BLS key mismatch"))
		}

		if err := verifySigSetVote(ctx, pl); err != nil {
			return err
		}

		ctx.SigSetVoteChan <- msg
		return nil
	default:
		return prerror.New(prerror.Low, fmt.Errorf("consensus: consensus payload has unrecognized ID %v",
			msg.Payload.Type()))
	}
}

func verifyBLSKey(ctx *user.Context, pubKeyEd, pubKeyBls []byte) bool {
	pk := hex.EncodeToString(pubKeyBls)
	return bytes.Equal(ctx.NodeBLS[pk], pubKeyEd)
}

func verifyReduction(ctx *user.Context, pl *consensusmsg.Reduction) *prerror.PrError {
	// Check BLS
	if err := ctx.BLSVerify(pl.PubKeyBLS, pl.BlockHash, pl.SigBLS); err != nil {
		return prerror.New(prerror.Low, errors.New("BLS verification failed"))
	}

	// Make sure they voted on an existing block
	blockHash := hex.EncodeToString(pl.BlockHash)
	if ctx.CandidateBlocks[blockHash] == nil {
		return prerror.New(prerror.Low, errors.New("block voted on not found"))
	}

	return nil
}

func verifyVoteSet(ctx *user.Context, voteSet []*consensusmsg.Vote, hash []byte, step uint8) *prerror.PrError {
	// A set should be of appropriate length, at least 2*0.75*len(committee)
	limit := 2.0 * 0.75 * float64(len(ctx.CurrentCommittee))
	if len(voteSet) < int(limit) {
		return prerror.New(prerror.Low, errors.New("vote set is too small"))
	}

	// Create committees
	committee, err := sortition.CreateCommittee(ctx.Round, ctx.W, step,
		uint8(len(ctx.CurrentCommittee)), ctx.Committee, ctx.NodeWeights)
	if err != nil {
		return prerror.New(prerror.High, err)
	}

	prevCommittee, err := sortition.CreateCommittee(ctx.Round, ctx.W, step-1,
		uint8(len(ctx.CurrentCommittee)), ctx.Committee, ctx.NodeWeights)
	if err != nil {
		return prerror.New(prerror.High, err)
	}

	for _, vote := range voteSet {
		// A set should only have votes for the designated hash
		if !bytes.Equal(hash, vote.Hash) {
			return prerror.New(prerror.Low, errors.New("vote is for wrong hash"))
		}

		// A set should only have votes from legitimate provisioners
		pkBLS := hex.EncodeToString(vote.PubKey)
		pkEd := hex.EncodeToString(ctx.NodeBLS[pkBLS])
		if ctx.NodeBLS[pkBLS] == nil || ctx.NodeWeights[pkEd] == 0 {
			return prerror.New(prerror.Low, errors.New("vote is from non-provisioner node"))
		}

		// A vote should be from the same step or the step before it
		if step != vote.Step && step != vote.Step+1 {
			return prerror.New(prerror.Low, errors.New("vote is from another phase"))
		}

		// A voting node should have been part of this or the previous committee
		if votes := sortition.Verify(committee, ctx.NodeBLS[pkBLS]); votes == 0 {
			if votes = sortition.Verify(prevCommittee, ctx.NodeBLS[pkBLS]); votes == 0 {
				return prerror.New(prerror.Low, errors.New("vote is not from committee"))
			}
		}

		// Signature verification
		if err := ctx.BLSVerify(vote.PubKey, vote.Hash, vote.Sig); err != nil {
			return prerror.New(prerror.Low, errors.New("BLS signature verification failed"))
		}
	}

	return nil
}

func verifySigSetCandidate(ctx *user.Context, pl *consensusmsg.SigSetCandidate, step uint8) *prerror.PrError {
	// We discard any deviating block hashes after the block reduction phase
	if !bytes.Equal(pl.WinningBlockHash, ctx.BlockHash) {
		return prerror.New(prerror.Low, errors.New("wrong block hash"))
	}

	return verifyVoteSet(ctx, pl.SignatureSet, pl.WinningBlockHash, step)
}

func verifySigSetVote(ctx *user.Context, pl *consensusmsg.SigSetVote) *prerror.PrError {
	// We discard any deviating block hashes after the block reduction phase
	if !bytes.Equal(pl.WinningBlockHash, ctx.BlockHash) {
		return prerror.New(prerror.Low, errors.New("wrong block hash"))
	}

	// Check BLS
	if err := ctx.BLSVerify(pl.PubKeyBLS, pl.SigSetHash, pl.SigBLS); err != nil {
		return prerror.New(prerror.Low, errors.New("BLS verification failed"))
	}

	// Vote must be for an existing vote set
	setStr := hex.EncodeToString(pl.SigSetHash)
	if ctx.AllVotes[setStr] == nil {
		return prerror.New(prerror.Low, errors.New("vote is for a non-existant vote set"))
	}

	return nil
}
