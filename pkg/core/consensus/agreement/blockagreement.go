package agreement

import (
	"encoding/hex"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/sortition"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// Block is the function that runs during the block reduction phase, used
// to collect vote sets and find a winning block. Block will run indefinitely
// until a decision is reached.
func Block(ctx *user.Context, c chan bool) {
	// Make a mapping of steps, pointing to a slice of vote sets.
	sets := make(map[uint8][][]*consensusmsg.Vote)

	for {
		select {
		case m := <-ctx.BlockAgreementChan:
			pl := m.Payload.(*consensusmsg.BlockAgreement)

			// Get amount of votes
			votes := sortition.Verify(ctx.CurrentCommittee, m.PubKey)

			// Add it to our set
			if sets[m.Step] == nil {
				sets[m.Step] = make([][]*consensusmsg.Vote, 0)
			}

			for i := uint8(0); i < votes; i++ {
				sets[m.Step] = append(sets[m.Step], pl.VoteSet)
			}

			// Check if we have exceeded the limit.
			limit := float64(len(ctx.CurrentCommittee)) * 0.75
			if len(sets[m.Step]) < int(limit) {
				break
			}

			// Populate certificate
			ctx.Certificate.BRPubKeys = make([][]byte, len(ctx.BlockVotes))
			for i := 0; i < len(ctx.BlockVotes); i++ {
				pkBLS := hex.EncodeToString(ctx.BlockVotes[i].PubKey)
				ctx.Certificate.BRPubKeys = append(ctx.Certificate.BRPubKeys, ctx.NodeBLS[pkBLS])
			}

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

// SendBlock will send out a block agreement message.
func SendBlock(ctx *user.Context) error {
	// Create payload and message
	pl, err := consensusmsg.NewBlockAgreement(ctx.BlockHash, ctx.BlockVotes)
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

	// Gossip message
	if err := ctx.SendMessage(ctx.Magic, msg); err != nil {
		return err
	}

	// Send it to our own agreement channel
	ctx.SigSetAgreementChan <- msg
	return nil
}
