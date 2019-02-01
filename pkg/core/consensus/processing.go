package consensus

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// Top-level message processing function for the consensus.
func processMsg(ctx *Context, msg *payload.MsgConsensus) (bool, uint64, error) {
	// Verify Ed25519 signature
	edMsg := new(bytes.Buffer)
	if err := msg.EncodeSignable(edMsg); err != nil {
		return false, 0, err
	}

	if !ctx.EDVerify(msg.PubKey, edMsg.Bytes(), msg.Signature) {
		return false, 0, nil
	}

	// Version check
	if ctx.Version != msg.Version {
		return false, 0, nil
	}

	// Check if we're on the same chain
	if !bytes.Equal(msg.PrevBlockHash, ctx.LastHeader.Hash) {
		return false, 0, nil
	}

	// Check if we're on the same round
	if ctx.Round != msg.Round {
		return false, 0, nil
	}

	// Proceed to more specific checks
	return verifyPayload(ctx, msg)
}

// Lower-level message processing function. This function determines the payload type,
// and applies the proper verification functions.
func verifyPayload(ctx *Context, msg *payload.MsgConsensus) (bool, uint64, error) {
	stake := ctx.NodeWeights[hex.EncodeToString(msg.PubKey)]

	switch msg.Payload.Type() {
	case consensusmsg.CandidateScoreID:
		// TODO: add actual verification code for score messages
		return true, 0, nil
	case consensusmsg.CandidateID:
		// Block was already verified upon reception, so we don't do anything else here.
		return true, 0, nil
	case consensusmsg.ReductionID:
		// Check if we're on the same step
		if ctx.Step != msg.Step {
			return false, 0, nil
		}

		pl := msg.Payload.(*consensusmsg.Reduction)
		votes, err := verifyReduction(ctx, pl, stake)
		if err != nil {
			return false, 0, err
		}

		if !verifyBLSKey(ctx, msg.PubKey, pl.PubKeyBLS) {
			return false, 0, nil
		}

		return true, votes, nil
	case consensusmsg.SetAgreementID:
		pl := msg.Payload.(*consensusmsg.SetAgreement)
		valid, err := verifyVoteSet(ctx, pl.VoteSet, pl.BlockHash, msg.Step)
		if err != nil {
			return false, 0, err
		}

		return valid, 0, nil
	case consensusmsg.SigSetCandidateID:
		pl := msg.Payload.(*consensusmsg.SigSetCandidate)
		valid, err := verifySigSetCandidate(ctx, pl, stake, msg.Step)
		if err != nil {
			return false, 0, err
		}

		return valid, stake, nil
	case consensusmsg.SigSetVoteID:
		pl := msg.Payload.(*consensusmsg.SigSetVote)
		if !verifyBLSKey(ctx, msg.PubKey, pl.PubKeyBLS) {
			return false, 0, nil
		}

		if !verifySigSetVote(ctx, pl, stake) {
			return false, 0, nil
		}

		return true, stake, nil
	default:
		return false, 0, fmt.Errorf("consensus: consensus payload has unrecognized ID %v",
			msg.Payload.Type())
	}
}

func verifyBLSKey(ctx *Context, pubKeyEd, pubKeyBls []byte) bool {
	pk := hex.EncodeToString(pubKeyBls)
	return bytes.Equal(ctx.NodeBLS[pk], pubKeyEd)
}

func verifyReduction(ctx *Context, pl *consensusmsg.Reduction, stake uint64) (uint64, error) {
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.Step,
	}

	// Make sure their score is valid, and calculate their amount of votes.
	votes, err := verifySortition(ctx, pl.Score, pl.PubKeyBLS, role, stake)
	if err != nil {
		return 0, err
	}

	if votes == 0 {
		return 0, nil
	}

	// Make sure they voted on an existing block
	blockHash := hex.EncodeToString(pl.BlockHash)
	if ctx.CandidateBlocks[blockHash] == nil {
		return 0, nil
	}

	return votes, nil
}

func verifyVoteSet(ctx *Context, voteSet []*consensusmsg.Vote, hash []byte, step uint8) (bool, error) {
	// A set should be of appropriate length, at least 2*0.75*len(provisioners)
	limit := 2.0 * 0.75 * float64(len(ctx.NodeWeights))
	if len(voteSet) < int(limit) {
		return false, nil
	}

	for _, vote := range voteSet {
		// A set should only have votes for the designated hash
		if !bytes.Equal(hash, vote.Hash) {
			return false, nil
		}

		// A set should only have votes from legitimate provisioners
		pkBLS := hex.EncodeToString(vote.PubKey)
		if ctx.NodeBLS[pkBLS] == nil {
			return false, nil
		}

		// A voter should have at least threshold stake amount
		pkEd := hex.EncodeToString(ctx.NodeBLS[pkBLS])
		if ctx.NodeWeights[pkEd] < MinimumStake {
			return false, nil
		}

		// A vote should be from the same step or the step before it
		if step != ctx.Step && step != ctx.Step-1 {
			return false, nil
		}

		// A vote's score should be verified
		role := &role{
			part:  "committee",
			round: ctx.Round,
			step:  ctx.Step,
		}

		votes, err := verifySortition(ctx, vote.Score, vote.PubKey, role, ctx.NodeWeights[pkEd])
		if err != nil {
			return false, err
		}

		if votes == 0 {
			return false, nil
		}

		// Signature verification
		if err := ctx.BLSVerify(vote.PubKey, vote.Hash, vote.Sig); err != nil {
			return false, nil
		}
	}

	return true, nil
}

func verifySigSetCandidate(ctx *Context, pl *consensusmsg.SigSetCandidate, stake uint64,
	step uint8) (bool, error) {
	// We discard any deviating block hashes after the block reduction phase
	if !bytes.Equal(pl.WinningBlockHash, ctx.BlockHash) {
		return false, nil
	}

	// Verify node sortition
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.Step,
	}

	votes, err := verifySortition(ctx, pl.Score, pl.PubKeyBLS, role, stake)
	if err != nil {
		return false, nil
	}

	if votes == 0 {
		return false, nil
	}

	return verifyVoteSet(ctx, pl.SignatureSet, pl.WinningBlockHash, step)
}

func verifySigSetVote(ctx *Context, pl *consensusmsg.SigSetVote, stake uint64) bool {
	// We discard any deviating block hashes after the block reduction phase
	if !bytes.Equal(pl.WinningBlockHash, ctx.BlockHash) {
		return false
	}

	// Check BLS
	if err := ctx.BLSVerify(pl.PubKeyBLS, pl.SigSetHash, pl.SigBLS); err != nil {
		return false
	}

	// Verify node sortition
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.Step,
	}

	votes, err := verifySortition(ctx, pl.Score, pl.PubKeyBLS, role, stake)
	if err != nil {
		return false
	}

	if votes == 0 {
		return false
	}

	// Vote must be for an existing vote set
	setStr := hex.EncodeToString(pl.SigSetHash)
	if ctx.AllVotes[setStr] == nil {
		return false
	}

	return true
}

func createSignature(ctx *Context, pl consensusmsg.Msg) ([]byte, error) {
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
