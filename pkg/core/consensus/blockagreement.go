package consensus

import (
	"encoding/hex"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// BlockAgreement is the function that runs during the block reduction phase, used
// to collect vote sets and find a winning block. BlockAgreement will run indefinitely
// until a decision is reached.
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

			// Add it to our collection
			pl := m.Payload.(*consensusmsg.SetAgreement)
			pkEd := hex.EncodeToString(m.PubKey)

			if sets[m.Step] == nil {
				sets[m.Step] = make(map[string][]*consensusmsg.Vote)
			}

			sets[m.Step][pkEd] = pl.VoteSet

			// Check if we have exceeded the limit
			limit := float64(len(ctx.NodeWeights)) * 0.75
			if len(sets[m.Step]) >= int(limit) {
				// Set our own vote set as the signature set for signature
				// set generation, which should follow after this
				pkEd := hex.EncodeToString([]byte(*ctx.Keys.EdPubKey))
				ctx.SigSetVotes = sets[m.Step][pkEd]

				// Populate certificate
				ctx.Certificate.BRPubKeys = make([][]byte, len(ctx.BlockVotes))
				ctx.Certificate.BRSortitionProofs = make([][]byte, len(ctx.BlockVotes))
				agSig := &bls.Signature{}
				if err := agSig.Decompress(ctx.BlockVotes[0].Sig); err != nil {
					// Log
					c <- false
					return
				}

				// Batch all the signatures together
				for i, vote := range ctx.BlockVotes {
					if i != 0 {
						sig := &bls.Signature{}
						if err := sig.Decompress(vote.Sig); err != nil {
							// Log
							c <- false
							return
						}

						agSig.Aggregate(sig)
					}

					ctx.Certificate.BRPubKeys[i] = vote.PubKey
					ctx.Certificate.BRSortitionProofs[i] = vote.Score

				}

				cSig := agSig.Compress()
				ctx.Certificate.BRBatchedSig = cSig
				ctx.Certificate.BRStep = m.Step
				c <- true
				return
			}
		}
	}
}
