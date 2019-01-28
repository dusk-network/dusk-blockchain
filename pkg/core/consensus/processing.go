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
		return true, 0, nil
	case consensusmsg.CandidateID:
		// Block was already verified upon reception, so we don't do anything else here.
		return true, 0, nil
	case consensusmsg.ReductionID:
		pl := msg.Payload.(*consensusmsg.Reduction)
		votes, err := verifyReduction(ctx, pl, stake)
		if err != nil {
			return false, 0, err
		}

		if !verifyBLSKey(ctx, msg.PubKey, pl.PubKeyBLS) {
			return false, 0, nil
		}

		return true, votes, nil
	case consensusmsg.AgreementID:
		pl := msg.Payload.(*consensusmsg.Agreement)
		votes, err := verifyAgreement(ctx, pl, stake)
		if err != nil {
			return false, 0, err
		}

		if !verifyBLSKey(ctx, msg.PubKey, pl.PubKeyBLS) {
			return false, 0, nil
		}

		return true, votes, nil
	// case consensusmsg.SetAgreementID:

	case consensusmsg.SigSetCandidateID:
		pl := msg.Payload.(*consensusmsg.SigSetCandidate)
		if !verifySigSetCandidate(ctx, pl, stake) {
			return false, 0, nil
		}

		return true, stake, nil
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
		return false, 0, fmt.Errorf("consensus: consensus payload has unrecognized ID %v", msg.Payload.Type())
	}
}

func verifyBLSKey(ctx *Context, pubKeyEd, pubKeyBls []byte) bool {
	pk := hex.EncodeToString(pubKeyEd)
	return bytes.Equal(ctx.NodeBLS[pk], pubKeyBls)
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

	return votes, nil
}

func verifyAgreement(ctx *Context, pl *consensusmsg.Agreement, stake uint64) (uint64, error) {
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

	return votes, nil
}

func verifySigSetCandidate(ctx *Context, pl *consensusmsg.SigSetCandidate, stake uint64) bool {
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.Step,
	}

	if !bytes.Equal(pl.WinningBlockHash, ctx.BlockHash) {
		return false
	}

	votes, err := verifySortition(ctx, pl.Score, pl.PubKeyBLS, role, stake)
	if err != nil {
		return false
	}

	if votes == 0 {
		return false
	}

	// Verify signature set

	return true
}

func verifySigSetVote(ctx *Context, pl *consensusmsg.SigSetVote, stake uint64) bool {
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.Step,
	}

	// Synchrony check
	if pl.Step != ctx.Step {
		return false
	}

	if !bytes.Equal(pl.WinningBlockHash, ctx.BlockHash) {
		return false
	}

	// Check BLS
	if err := ctx.BLSVerify(pl.PubKeyBLS, pl.SigSetHash, pl.SigBLS); err != nil {
		return false
	}

	votes, err := verifySortition(ctx, pl.Score, pl.PubKeyBLS, role, stake)
	if err != nil {
		return false
	}

	if votes == 0 {
		return false
	}

	return true
}

func createSignature(ctx *Context, pl consensusmsg.Msg) ([]byte, error) {
	edMsg := make([]byte, 12)
	binary.LittleEndian.PutUint32(edMsg[0:], ctx.Version)
	binary.LittleEndian.PutUint64(edMsg[4:], ctx.Round)
	edMsg = append(edMsg, ctx.LastHeader.Hash...)
	edMsg = append(edMsg, byte(pl.Type()))
	buf := new(bytes.Buffer)
	if err := pl.Encode(buf); err != nil {
		return nil, err
	}

	edMsg = append(edMsg, buf.Bytes()...)
	return ctx.EDSign(ctx.Keys.EdSecretKey, edMsg), nil
}
