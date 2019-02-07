package consensus

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/sortition"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// ProcessMsg is a top-level message processing function for the consensus.
func ProcessMsg(ctx *Context, msg *payload.MsgConsensus) (uint8, *prerror.PrError) {
	// Verify Ed25519 signature
	edMsg := new(bytes.Buffer)
	if err := msg.EncodeSignable(edMsg); err != nil {
		return 0, prerror.New(prerror.High, err)
	}

	if !ctx.EDVerify(msg.PubKey, edMsg.Bytes(), msg.Signature) {
		return 0, prerror.New(prerror.Low, errors.New("ed25519 verification failed"))
	}

	// Check if this node is eligible to vote, and see how many votes they get
	votes := sortition.Verify(ctx.CurrentCommittee, msg.PubKey)
	if votes == 0 {
		return 0, prerror.New(prerror.Low, errors.New("node is not included in current committee"))
	}

	// Version check
	if ctx.Version != msg.Version {
		return 0, prerror.New(prerror.Low, errors.New("version mismatch"))
	}

	// Check if we're on the same chain
	if !bytes.Equal(msg.PrevBlockHash, ctx.LastHeader.Hash) {
		return 0, prerror.New(prerror.Low, errors.New("voter is on a different chain"))
	}

	// Check if we're on the same round
	if ctx.Round != msg.Round {
		return 0, prerror.New(prerror.Low, errors.New("round mismatch"))
	}

	// Proceed to more specific checks
	return votes, verifyPayload(ctx, msg)
}

// Lower-level message processing function. This function determines the payload type,
// and applies the proper verification functions.
func verifyPayload(ctx *Context, msg *payload.MsgConsensus) *prerror.PrError {
	switch msg.Payload.Type() {
	case consensusmsg.CandidateScoreID:
		// TODO: add actual verification code for score messages
		return nil
	case consensusmsg.CandidateID:
		// Block was already verified upon reception, so we don't do anything else here.
		return nil
	case consensusmsg.ReductionID:
		// Check if we're on the same step
		if ctx.Step != msg.Step {
			return prerror.New(prerror.Low, errors.New("step mismatch"))
		}

		pl := msg.Payload.(*consensusmsg.Reduction)
		if err := verifyReduction(ctx, pl); err != nil {
			return err
		}

		if !verifyBLSKey(ctx, msg.PubKey, pl.PubKeyBLS) {
			return prerror.New(prerror.Low, errors.New("BLS key mismatch"))
		}

		return nil
	case consensusmsg.SetAgreementID:
		pl := msg.Payload.(*consensusmsg.SetAgreement)
		if err := verifyVoteSet(ctx, pl.VoteSet, pl.BlockHash, msg.Step); err != nil {
			return err
		}

		return nil
	case consensusmsg.SigSetCandidateID:
		pl := msg.Payload.(*consensusmsg.SigSetCandidate)
		if err := verifySigSetCandidate(ctx, pl, msg.Step); err != nil {
			return err
		}

		return nil
	case consensusmsg.SigSetVoteID:
		pl := msg.Payload.(*consensusmsg.SigSetVote)
		if !verifyBLSKey(ctx, msg.PubKey, pl.PubKeyBLS) {
			return prerror.New(prerror.Low, errors.New("BLS key mismatch"))
		}

		if err := verifySigSetVote(ctx, pl); err != nil {
			return err
		}

		return nil
	default:
		return prerror.New(prerror.Low, fmt.Errorf("consensus: consensus payload has unrecognized ID %v",
			msg.Payload.Type()))
	}
}

func verifyBLSKey(ctx *Context, pubKeyEd, pubKeyBls []byte) bool {
	pk := hex.EncodeToString(pubKeyBls)
	return bytes.Equal(ctx.NodeBLS[pk], pubKeyEd)
}

func verifyReduction(ctx *Context, pl *consensusmsg.Reduction) *prerror.PrError {
	// Make sure they voted on an existing block
	blockHash := hex.EncodeToString(pl.BlockHash)
	if ctx.CandidateBlocks[blockHash] == nil {
		return prerror.New(prerror.Low, errors.New("block voted on not found"))
	}

	return nil
}

func verifyVoteSet(ctx *Context, voteSet []*consensusmsg.Vote, hash []byte, step uint8) *prerror.PrError {
	// A set should be of appropriate length, at least 2*0.75*len(committee)
	limit := 2.0 * 0.75 * float64(len(ctx.CurrentCommittee))
	if len(voteSet) < int(limit) {
		return prerror.New(prerror.Low, errors.New("vote set is too small"))
	}

	// Create committee from previous step
	prevCommittee, err := sortition.Deterministic(ctx.Round, ctx.W, ctx.Step-1, uint8(len(ctx.CurrentCommittee)),
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
		if ctx.NodeBLS[pkBLS] == nil {
			return prerror.New(prerror.Low, errors.New("vote is from non-provisioner node"))
		}

		// A vote should be from the same step or the step before it
		if step != ctx.Step && step != ctx.Step-1 {
			return prerror.New(prerror.Low, errors.New("vote is from another phase"))
		}

		// A voting node should have been part of this or the previous committee
		if votes := sortition.Verify(ctx.CurrentCommittee, ctx.NodeBLS[pkBLS]); votes == 0 {
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

func verifySigSetCandidate(ctx *Context, pl *consensusmsg.SigSetCandidate, step uint8) *prerror.PrError {
	// We discard any deviating block hashes after the block reduction phase
	if !bytes.Equal(pl.WinningBlockHash, ctx.BlockHash) {
		return prerror.New(prerror.Low, errors.New("wrong block hash"))
	}

	return verifyVoteSet(ctx, pl.SignatureSet, pl.WinningBlockHash, step)
}

func verifySigSetVote(ctx *Context, pl *consensusmsg.SigSetVote) *prerror.PrError {
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

// CreateSignature will return the byte representation of a consensus message that
// is signed with Ed25519.
func CreateSignature(ctx *Context, pl consensusmsg.Msg) ([]byte, error) {
	edMsg := make([]byte, 12)
	binary.LittleEndian.PutUint32(edMsg[0:], ctx.Version)
	binary.LittleEndian.PutUint64(edMsg[4:], ctx.Round)
	edMsg = append(edMsg, ctx.LastHeader.Hash...)
	edMsg = append(edMsg, byte(ctx.Step))
	edMsg = append(edMsg, byte(pl.Type()))
	buf := new(bytes.Buffer)
	if err := pl.Encode(buf); err != nil {
		return nil, err
	}

	edMsg = append(edMsg, buf.Bytes()...)
	return ctx.EDSign(ctx.Keys.EdSecretKey, edMsg), nil
}
