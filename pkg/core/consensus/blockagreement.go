package consensus

import (
	"encoding/hex"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// BlockAgreement is the function that runs during the block reduction phase, used
// to collect vote sets and find a winning block.
func BlockAgreement(ctx *Context, c chan bool) {
	// Make a mapping of steps, pointing to a mapping of an Ed25519 public keys and
	// vote sets.
	sets := make(map[uint8]map[string][]*consensusmsg.Vote)

	for {
		select {
		case m := <-ctx.SetAgreementChan:
			// Process received message
			valid, _, err := processMsg(ctx, m)
			if err != nil {
				// Log
				c <- false
				return
			}

			// Discard if it's invalid
			if !valid {
				break
			}

			// Add it to our counter
			pl := m.Payload.(*consensusmsg.SetAgreement)
			pkEd := hex.EncodeToString(m.PubKey)

			if sets[m.Step] == nil {
				sets[m.Step] = make(map[string][]*consensusmsg.Vote)
			}

			sets[m.Step][pkEd] = pl.VoteSet

			// Check if we have exceeded the limit
			if uint64(len(sets[m.Step])) >= ctx.VoteLimit {
				// Set our own vote set as the signature set for signature
				// set generation, which should follow after this
				pkEd := hex.EncodeToString([]byte(*ctx.Keys.EdPubKey))
				ctx.SigSetVotes = sets[m.Step][pkEd]

				// Set certificate values
				// TODO: batched sigs
				// ctx.Certificate.BRStep = ctx.Step
				// TODO: pubkeys
				// TODO: sortition proofs
				c <- true
				return
			}
		}
	}
}
