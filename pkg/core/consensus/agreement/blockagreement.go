package agreement

import (
	"encoding/hex"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"
)

// Block is the function that runs during the block reduction phase, used
// to collect vote sets and find a winning block. Block will run indefinitely
// until a decision is reached.
func Block(ctx *user.Context, c chan bool) {
	// Make a mapping of steps, pointing to a mapping of an Ed25519 public keys and
	// vote sets.
	sets := make(map[uint8]map[string][]*consensusmsg.Vote)

	for {
		select {
		case m := <-ctx.SetAgreementChan:
			// Process received message
			_, err := msg.Process(ctx, m)
			if err != nil {
				if err.Priority == prerror.High {
					// Log
					c <- false
					return
				}

				// Discard if invalid
				break
			}

			// Add it to our collection
			pl := m.Payload.(*consensusmsg.SetAgreement)
			pkEd := hex.EncodeToString(m.PubKey)

			if sets[m.Step] == nil {
				sets[m.Step] = make(map[string][]*consensusmsg.Vote)
			}

			sets[m.Step][pkEd] = pl.VoteSet

			// Check if we have exceeded the limit.
			limit := float64(len(ctx.NodeWeights)) * 0.75
			if len(sets[m.Step]) < int(limit) {
				break
			}

			// Set our own vote set as the signature set for signature
			// set generation, which should follow after this
			ctxPkEd := hex.EncodeToString([]byte(*ctx.Keys.EdPubKey))
			ctx.SigSetVotes = sets[m.Step][ctxPkEd]

			// Populate certificate
			ctx.Certificate.BRPubKeys = make([][]byte, len(ctx.BlockVotes))
			agSig := &bls.Signature{}
			if err := agSig.Decompress(ctx.BlockVotes[0].Sig); err != nil {
				// Log
				c <- false
				return
			}

			// Batch all the signatures together
			for i, vote := range ctx.BlockVotes {
				// Skip the one we already got (agSig)
				if i == 0 {
					continue
				}

				sig := &bls.Signature{}
				if err := sig.Decompress(vote.Sig); err != nil {
					// Log
					c <- false
					return
				}

				agSig.Aggregate(sig)
				ctx.Certificate.BRPubKeys[i] = vote.PubKey
			}

			cSig := agSig.Compress()
			ctx.Certificate.BRBatchedSig = cSig
			ctx.Certificate.BRStep = m.Step
			c <- true
			return
		}
	}
}
