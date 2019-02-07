package collection

import (
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"
)

// SigSet will collect the signature set with highest score, and set our context value
// to the winner.
func SigSet(ctx *user.Context) error {
	// Keep track of those who have voted
	voters := make(map[string]bool)
	pk := hex.EncodeToString([]byte(*ctx.Keys.EdPubKey))

	// Log our own key
	voters[pk] = true

	// Initialize container for all vote sets, and add our own
	ctx.AllVotes = make(map[string][]*consensusmsg.Vote)
	sigSetHash, err := msg.HashVotes(ctx.SigSetVotes)
	if err != nil {
		return err
	}

	ctx.AllVotes[hex.EncodeToString(sigSetHash)] = ctx.SigSetVotes
	highest := ctx.Weight

	// Start timer
	timer := time.NewTimer(user.StepTime * (time.Duration(ctx.Multiplier) * time.Second))

	for {
		select {
		case <-timer.C:
			return nil
		case m := <-ctx.SigSetCandidateChan:
			pl := m.Payload.(*consensusmsg.SigSetCandidate)
			pkEd := hex.EncodeToString(m.PubKey)

			// Check if this node's signature set is already recorded
			if voters[pkEd] {
				break
			}

			// Verify the message
			if _, prErr := msg.Process(ctx, m); prErr != nil {
				if prErr.Priority == prerror.High {
					return prErr.Err
				}

				// Discard if it's invalid
				break
			}

			// Log information
			voters[pkEd] = true
			setHash, err := msg.HashVotes(pl.SignatureSet)
			if err != nil {
				return err
			}

			ctx.AllVotes[hex.EncodeToString(setHash)] = pl.SignatureSet
			stake := ctx.NodeWeights[pkEd]

			// If the stake is higher than our current one, replace
			if stake > highest {
				highest = stake
				ctx.SigSetVotes = pl.SignatureSet
			}
		}
	}
}
