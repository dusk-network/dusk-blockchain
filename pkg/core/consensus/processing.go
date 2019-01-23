package consensus

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// Message processing function for the consensus.
func processMsg(ctx *Context, msg *payload.MsgConsensus) (bool, uint64, error) {
	// Verify Ed25519 signature
	edMsg := new(bytes.Buffer)
	if err := msg.EncodeSignable(edMsg); err != nil {
		return false, 0, err
	}

	if !ctx.EDVerify(msg.PubKey, edMsg.Bytes(), msg.Signature) {
		return false, 0, nil
	}

	// Check if we're on the same chain
	if !bytes.Equal(msg.PrevBlockHash, ctx.LastHeader.Hash) {
		return false, 0, nil
	}

	// Return the payload for further processing
	return verifyPayload(ctx, msg)
}

func verifyPayload(ctx *Context, msg *payload.MsgConsensus) (bool, uint64, error) {
	stake := ctx.NodeWeights[hex.EncodeToString(msg.PubKey)]

	switch msg.Payload.Type() {
	case consensusmsg.CandidateScoreID:
		return true, 0, nil
	case consensusmsg.CandidateID:
		return true, 0, nil
	case consensusmsg.ReductionID:
		pl := msg.Payload.(*consensusmsg.Reduction)
		votes, err := verifyReduction(ctx, pl, stake)
		if err != nil {
			return false, 0, err
		}

		return true, votes, nil
	case consensusmsg.AgreementID:
		pl := msg.Payload.(*consensusmsg.Agreement)
		votes, err := verifyAgreement(ctx, pl, stake)
		if err != nil {
			return false, 0, err
		}

		return true, votes, nil
	// case consensusmsg.SetAgreementID:

	case consensusmsg.SigSetCandidateID:
		return true, stake, nil
	case consensusmsg.SigSetVoteID:
		pl := msg.Payload.(*consensusmsg.SigSetVote)
		return verifySigSetVote(ctx, pl), stake, nil
	default:
		return false, 0, fmt.Errorf("consensus: consensus payload has unrecognized ID %v", msg.Payload.Type())
	}
}

func verifyReduction(ctx *Context, pl *consensusmsg.Reduction, stake uint64) (uint64, error) {
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.step,
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
		step:  ctx.step,
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

func verifySigSetVote(ctx *Context, pl *consensusmsg.SigSetVote) bool {
	// Synchrony check
	if pl.Step != ctx.step {
		return false
	}

	// Check BLS
	if err := ctx.BLSVerify(pl.PubKeyBLS, pl.SignatureSet, pl.SigBLS); err != nil {
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
