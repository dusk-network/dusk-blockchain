package collection

import (
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// SignatureSet will collect the signature set with highest score, and set our context value
// to the winner.
func SignatureSet(ctx *user.Context) error {
	var highest uint64

	// Start timer
	timer := time.NewTimer(user.StepTime * (time.Duration(ctx.Multiplier)))

	// Process queue
	if err := msg.ProcessQueue(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-timer.C:
			return nil
		case m := <-ctx.SigSetCandidateChan:
			pl := m.Payload.(*consensusmsg.SigSetCandidate)
			pkEd := hex.EncodeToString(m.PubKey)
			stake := ctx.NodeWeights[pkEd]

			// If the stake is higher than our current one, replace
			if stake > highest {
				highest = stake
				ctx.SigSetVotes = pl.SignatureSet
				setHash, err := ctx.HashVotes(pl.SignatureSet)
				if err != nil {
					return err
				}

				ctx.SigSetHash = setHash

				// Gossip it to the rest of the network
				ctx.SendMessage(ctx.Magic, m)
			}
		}
	}
}
