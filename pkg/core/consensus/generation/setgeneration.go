package generation

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// SignatureSet will generate a signature set message and gossip it.
func SignatureSet(ctx *user.Context) error {
	// Create our own signature set candidate message
	pl, err := consensusmsg.NewSigSetCandidate(ctx.BlockHash, ctx.SigSetVotes,
		ctx.Certificate.BRStep)
	if err != nil {
		return err
	}

	sigEd, err := ctx.CreateSignature(pl)
	if err != nil {
		return err
	}

	msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
		ctx.Step, sigEd, []byte(*ctx.Keys.EdPubKey), pl)
	if err != nil {
		return err
	}

	// Gossip msg
	ctx.SigSetCandidateChan <- msg

	return nil
}
