package consensus

import (
	"bytes"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// SignatureSetGeneration will generate a signature set message, gossip it, and
// then collect all other messages, then retaining the most voted set for the
// signature set reduction phase.
func SignatureSetGeneration(ctx *Context) error {
	// Create our own signature set candidate message
	pl, err := consensusmsg.NewSigSetCandidate(ctx.BlockHash, ctx.SigSetVotes,
		ctx.Keys.BLSPubKey.Marshal(), ctx.Score)
	if err != nil {
		return err
	}

	sigEd, err := createSignature(ctx, pl)
	if err != nil {
		return err
	}

	msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
		ctx.Step, sigEd, []byte(*ctx.Keys.EdPubKey), pl)
	if err != nil {
		return err
	}

	// Gossip msg
	if err := ctx.SendMessage(ctx.Magic, msg); err != nil {
		return err
	}

	// Collect signature set with highest score, and set our context value
	// to the winner.
	var voters [][]byte
	voters = append(voters, []byte(*ctx.Keys.EdPubKey))
	highest := ctx.weight
	timer := time.NewTimer(stepTime)

	for {
	out:
		select {
		case <-timer.C:
			return nil
		case m := <-ctx.SigSetCandidateChan:
			pl := m.Payload.(*consensusmsg.SigSetCandidate)

			// Check if this node's signature set is already recorded
			for _, voter := range voters {
				if bytes.Equal(voter, m.PubKey) {
					break out
				}
			}

			// Verify the message
			valid, stake, err := processMsg(ctx, m)
			if err != nil {
				return err
			}

			// Discard if it's invalid
			if !valid {
				break
			}

			// If the stake is higher than our current one, replace
			if stake > highest {
				highest = stake
				ctx.SigSetVotes = pl.SignatureSet
			}
		}
	}
}
