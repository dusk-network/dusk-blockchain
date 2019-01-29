package consensus

import (
	"encoding/hex"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// SignatureSetAgreement is the function that runs during the signature set reduction
// phase, usednto collect vote sets and find a winning signature set.
func SignatureSetAgreement(ctx *Context, c chan bool) {
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
			sets[m.Step][pkEd] = pl.VoteSet

			// Check if we have exceeded the limit
			if uint64(len(sets[m.Step])) >= ctx.VoteLimit {
				c <- true
				return
			}
		}
	}
}
