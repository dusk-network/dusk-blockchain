package agreement

import (
	"encoding/hex"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/sortition"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// SignatureSet is the function that runs during the signature set reduction
// phase, used to collect vote sets and find a winning signature set.
func SignatureSet(ctx *user.Context, c chan bool) {
	// Keep track of all nodes who have voted
	voters := make(map[string]bool)

	// Make a mapping of steps, pointing to a slice of vote sets.
	sets := make(map[uint8][][]*consensusmsg.Vote)

	for {
		select {
		case m := <-ctx.SigSetAgreementChan:
			pl := m.Payload.(*consensusmsg.SigSetAgreement)
			pkEd := hex.EncodeToString(m.PubKey)

			// Check if this node's vote is already recorded
			if voters[pkEd] {
				break
			}

			voters[pkEd] = true

			// Get amount of votes
			votes := sortition.Verify(ctx.CurrentCommittee, m.PubKey)

			// Add it to our set
			if sets[m.Step] == nil {
				sets[m.Step] = make([][]*consensusmsg.Vote, 0)
			}

			for i := uint8(0); i < votes; i++ {
				sets[m.Step] = append(sets[m.Step], pl.VoteSet)
			}

			// Gossip the message
			if err := ctx.SendMessage(ctx.Magic, m); err != nil {
				// Log
				c <- false
				return
			}

			// Check if we have exceeded the limit
			limit := float64(len(ctx.CurrentCommittee)) * 0.75
			if len(sets[m.Step]) < int(limit) {
				break
			}

			// // Populate certificate
			// ctx.Certificate.SRPubKeys = make([][]byte, len(pl.VoteSet))
			// for i := 0; i < len(pl.VoteSet); i++ {
			// 	pkBLS := hex.EncodeToString(pl.VoteSet[i].PubKey)
			// 	ctx.Certificate.SRPubKeys = append(ctx.Certificate.SRPubKeys, ctx.NodeBLS[pkBLS])
			// }

			// agSig := &bls.Signature{}
			// if err := agSig.Decompress(ctx.SigSetVotes[0].Sig); err != nil {
			// 	// Log
			// 	c <- false
			// 	return
			// }

			// // Batch all the signatures together
			// for i, vote := range ctx.SigSetVotes {
			// 	if i != 0 {
			// 		continue
			// 	}

			// 	sig := &bls.Signature{}
			// 	if err := sig.Decompress(vote.Sig); err != nil {
			// 		// Log
			// 		c <- false
			// 		return
			// 	}

			// 	agSig.Aggregate(sig)
			// 	ctx.Certificate.SRPubKeys[i] = vote.PubKey
			// }

			// cSig := agSig.Compress()
			// ctx.Certificate.SRBatchedSig = cSig
			// ctx.Certificate.SRStep = ctx.Step
			c <- true
			return
		}
	}
}

// SendSigSet will send out a signature set agreement message.
func SendSigSet(ctx *user.Context) error {
	// Create payload, signature and message
	hash, err := ctx.HashVotes(ctx.SigSetVotes)
	if err != nil {
		return err
	}

	pl, err := consensusmsg.NewSigSetAgreement(ctx.BlockHash, hash, ctx.SigSetVotes,
		ctx.Certificate.BRStep)
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
