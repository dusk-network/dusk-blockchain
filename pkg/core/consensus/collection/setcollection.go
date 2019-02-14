package collection

import (
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// SignatureSet will collect the signature set with highest score, and set our context value
// to the winner.
func SignatureSet(ctx *user.Context) error {
	// Keep track of those who have voted
	voters := make(map[string]bool)
	pk := hex.EncodeToString([]byte(*ctx.Keys.EdPubKey))

	// Log our own key
	voters[pk] = true

	// Initialize container for all vote sets, and add our own
	ctx.AllVotes = make(map[string][]*consensusmsg.Vote)
	sigSetHash, err := ctx.HashVotes(ctx.SigSetVotes)
	if err != nil {
		return err
	}

	ctx.AllVotes[hex.EncodeToString(sigSetHash)] = ctx.SigSetVotes
	highest := ctx.Weight

	// Start timer
	timer := time.NewTimer(user.StepTime * (time.Duration(ctx.Multiplier)))

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

			// Log information
			voters[pkEd] = true
			setHash, err := ctx.HashVotes(pl.SignatureSet)
			if err != nil {
				return err
			}

			ctx.AllVotes[hex.EncodeToString(setHash)] = pl.SignatureSet
			stake := ctx.NodeWeights[pkEd]

			// If the stake is higher than our current one, replace
			if stake > highest {
				highest = stake
				ctx.SigSetVotes = pl.SignatureSet

				// Gossip it to the rest of the network
				ctx.SendMessage(ctx.Magic, m)
			}
		}
	}
}
