package agreement

import (
	"encoding/hex"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// SignatureSet is the function that runs during the signature set reduction
// phase, used to collect vote sets and find a winning signature set.
func SignatureSet(ctx *user.Context, c chan bool) {
	// Make a mapping of steps, pointing to a mapping of an Ed25519 public keys and
	// vote sets.
	sets := make(map[uint8]map[string][]*consensusmsg.Vote)

	for {
		select {
		case m := <-ctx.SetAgreementChan:
			pl := m.Payload.(*consensusmsg.SetAgreement)

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

			// Add it to our counter
			pkEd := hex.EncodeToString(m.PubKey)

			if sets[m.Step] == nil {
				sets[m.Step] = make(map[string][]*consensusmsg.Vote)
			}

			sets[m.Step][pkEd] = pl.VoteSet

			// Check if we have exceeded the limit
			limit := float64(len(ctx.NodeWeights)) * 0.75
			if len(sets[m.Step]) < int(limit) {
				break
			}

			// Populate certificate
			ctx.Certificate.SRPubKeys = make([][]byte, len(ctx.SigSetVotes))
			agSig := &bls.Signature{}
			if err := agSig.Decompress(ctx.SigSetVotes[0].Sig); err != nil {
				// Log
				c <- false
				return
			}

			// Batch all the signatures together
			for i, vote := range ctx.SigSetVotes {
				if i != 0 {
					continue
				}

				sig := &bls.Signature{}
				if err := sig.Decompress(vote.Sig); err != nil {
					// Log
					c <- false
					return
				}

				agSig.Aggregate(sig)
				ctx.Certificate.SRPubKeys[i] = vote.PubKey
			}

			cSig := agSig.Compress()
			ctx.Certificate.SRBatchedSig = cSig
			ctx.Certificate.SRStep = ctx.Step
			c <- true
			return
		}
	}
}

// SendSet will send out a set agreement message with the passed vote set.
func SendSet(ctx *user.Context, votes []*consensusmsg.Vote) error {
	// Create payload, signature and message
	pl, err := consensusmsg.NewSetAgreement(ctx.BlockHash, votes)
	if err != nil {
		return err
	}

	sigEd, err := msg.CreateSignature(ctx, pl)
	if err != nil {
		return err
	}

	msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
		ctx.Step, sigEd, []byte(*ctx.Keys.EdPubKey), pl)
	if err != nil {
		return err
	}

	// Gossip message
	if err := ctx.SendMessage(ctx.Magic, msg); err != nil {
		return err
	}

	// Send it to our own agreement channel
	ctx.SetAgreementChan <- msg
	return nil
}
