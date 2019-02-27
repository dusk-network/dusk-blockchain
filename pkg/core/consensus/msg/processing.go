package msg

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"sync/atomic"

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

	// Version check
	if ctx.Version != msg.Version {
		return prerror.New(prerror.Low, errors.New("version mismatch"))
	}

	// Check if we're on the same round
	if ctx.Round > msg.Round {
		return prerror.New(prerror.Low, errors.New("round mismatch"))
	}

	switch msg.ID {
	case consensusmsg.BlockAgreementID, consensusmsg.BlockReductionID,
		consensusmsg.CandidateID, consensusmsg.CandidateScoreID:
		if ctx.Round < msg.Round || atomic.LoadUint32(&ctx.BlockStep) < msg.Step {
			ctx.BlockQueue.Put(msg.Round, msg.Step, msg)
			return nil
		}
	case consensusmsg.SigSetAgreementID, consensusmsg.SigSetCandidateID,
		consensusmsg.SigSetReductionID:
		if ctx.Round < msg.Round || atomic.LoadUint32(&ctx.SigSetStep) < msg.Step {
			ctx.SigSetQueue.Put(msg.Round, msg.Step, msg)
			return nil
		}
	}

	// Check if we're on the same chain
	if !bytes.Equal(msg.PrevBlockHash, ctx.LastHeader.Hash) {
		return prerror.New(prerror.Low, errors.New("voter is on a different chain"))
	}

	// Proceed to more specific checks
	return verifyPayload(ctx, msg)
}

// ProcessBlockQueue will process messages in the queue
func ProcessBlockQueue(ctx *user.Context) *prerror.PrError {
	msgs := ctx.BlockQueue.Get(ctx.Round, atomic.LoadUint32(&ctx.BlockStep))
	if msgs == nil {
		return nil
	}

	for _, m := range msgs {
		err := Process(ctx, m)
		if err != nil && err.Priority == prerror.High {
			return err
		}
	}

	return nil
}

// ProcessSigSetQueue will process messages in the queue
func ProcessSigSetQueue(ctx *user.Context) *prerror.PrError {
	msgs := ctx.SigSetQueue.Get(ctx.Round, atomic.LoadUint32(&ctx.SigSetStep))
	if msgs == nil {
		return nil
	}

	for _, m := range msgs {
		err := Process(ctx, m)
		if err != nil && err.Priority == prerror.High {
			return err
		}
	}

	return nil
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
	case consensusmsg.BlockReductionID:
		// Check if we're on the same step
		if atomic.LoadUint32(&ctx.BlockStep) > msg.Step {
			return prerror.New(prerror.Low, errors.New("step mismatch"))
		}

		// Verify sortition
		if err := verifySortition(ctx, msg); err != nil {
			return err
		}

		pl := msg.Payload.(*consensusmsg.BlockReduction)
		if !verifyBLSKey(ctx, msg.PubKey, pl.PubKeyBLS) {
			return prerror.New(prerror.Low, errors.New("BLS key mismatch"))
		}

		// Check BLS
		if err := ctx.BLSVerify(pl.PubKeyBLS, pl.BlockHash, pl.SigBLS); err != nil {
			return prerror.New(prerror.Low, errors.New("BLS verification failed"))
		}

		ctx.BlockReductionChan <- msg
		return nil
	case consensusmsg.BlockAgreementID:
		// Verify sortition
		if err := verifySortition(ctx, msg); err != nil {
			return err
		}

		pl := msg.Payload.(*consensusmsg.BlockAgreement)
		if err := verifyVoteSet(ctx, pl.VoteSet, pl.BlockHash, msg.Step); err != nil {
			return err
		}

		ctx.BlockAgreementChan <- msg
		return nil
	case consensusmsg.SigSetCandidateID:
		pl := msg.Payload.(*consensusmsg.SigSetCandidate)
		if err := verifyVoteSet(ctx, pl.SignatureSet, pl.WinningBlockHash, pl.Step); err != nil {
			return err
		}

		ctx.SigSetCandidateChan <- msg
		return nil
	case consensusmsg.SigSetReductionID:
		// Check if we're on the same step
		if atomic.LoadUint32(&ctx.SigSetStep) > msg.Step {
			return prerror.New(prerror.Low, errors.New("step mismatch"))
		}

		// Verify sortition
		if err := verifySortition(ctx, msg); err != nil {
			return err
		}

		pl := msg.Payload.(*consensusmsg.SigSetReduction)
		if !verifyBLSKey(ctx, msg.PubKey, pl.PubKeyBLS) {
			return prerror.New(prerror.Low, errors.New("BLS key mismatch"))
		}

		if err := verifySigSetReduction(ctx, pl); err != nil {
			return err
		}

		ctx.SigSetReductionChan <- msg
		return nil
	case consensusmsg.SigSetAgreementID:
		// Verify sortition
		if err := verifySortition(ctx, msg); err != nil {
			return err
		}

		pl := msg.Payload.(*consensusmsg.SigSetAgreement)

		// We discard any deviating block hashes after the block reduction phase
		if !bytes.Equal(pl.BlockHash, ctx.WinningBlockHash) {
			return prerror.New(prerror.Low, errors.New("wrong block hash"))
		}

		if err := verifyVoteSet(ctx, pl.VoteSet, pl.SetHash, pl.Step); err != nil {
			return err
		}

		ctx.SigSetAgreementChan <- msg
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

func verifySortition(ctx *user.Context, msg *payload.MsgConsensus) *prerror.PrError {
	// Check what step this message is from, and reconstruct the committee
	// for that step.
	committee, err := sortition.CreateCommittee(ctx.Round, ctx.W, msg.Step,
		ctx.Committee, ctx.NodeWeights)
	if err != nil {
		return prerror.New(prerror.High, err)
	}

	// Check if this node is eligible to vote in this step
	votes := sortition.Verify(committee, msg.PubKey)
	if votes == 0 {
		return prerror.New(prerror.Low, errors.New("node is not included in committee"))
	}

	return nil
}

func verifyVoteSet(ctx *user.Context, voteSet []*consensusmsg.Vote, hash []byte,
	step uint32) *prerror.PrError {
	// A set should be of appropriate length, at least 2*0.75*len(committee)
	limit := int(2 * 0.75 * float64(len(ctx.Committee)))
	if limit > 100 {
		limit = 100
	}

	if len(voteSet) < limit {
		return prerror.New(prerror.Low, errors.New("vote set is too small"))
	}

	// Create committees
	committee, err := sortition.CreateCommittee(ctx.Round, ctx.W, step,
		ctx.Committee, ctx.NodeWeights)
	if err != nil {
		return prerror.New(prerror.High, err)
	}

	prevCommittee, err := sortition.CreateCommittee(ctx.Round, ctx.W, step-1,
		ctx.Committee, ctx.NodeWeights)
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
		sigCopy := make([]byte, len(vote.Sig))
		copy(sigCopy, vote.Sig)
		if err := ctx.BLSVerify(vote.PubKey, vote.Hash, sigCopy); err != nil {
			return prerror.New(prerror.Low, errors.New("BLS signature verification failed"))
		}
	}

	return nil
}

func verifySigSetReduction(ctx *user.Context, pl *consensusmsg.SigSetReduction) *prerror.PrError {
	// We discard any deviating block hashes after the block reduction phase
	if !bytes.Equal(pl.WinningBlockHash, ctx.WinningBlockHash) {
		return prerror.New(prerror.Low, errors.New("wrong block hash"))
	}

	// Check BLS
	if err := ctx.BLSVerify(pl.PubKeyBLS, pl.SigSetHash, pl.SigBLS); err != nil {
		return prerror.New(prerror.Low, errors.New("BLS verification failed"))
	}

	return nil
}
