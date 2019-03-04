package agreement

import (
	"bytes"
	"encoding/hex"
	"sync/atomic"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/sortition"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// SignatureSet is the function that runs during the signature set reduction
// phase, used to collect vote sets and find a winning signature set.
func SignatureSet(ctx *user.Context, c chan bool) {
	// Store our sets on every step for certificate generation
	sets := make(map[uint32][]*consensusmsg.Vote)

	// Make a counter to keep track of how many votes have been cast in a step
	counter := make(map[uint32]int)

	// Make a map to keep track if a node has voted in a certain step
	voted := make(map[uint32]map[string]bool)

	for {
		select {
		case m := <-ctx.SigSetAgreementChan:
			if m.Round != ctx.Round {
				break
			}

			pl := m.Payload.(*consensusmsg.SigSetAgreement)
			pkEd := hex.EncodeToString(m.PubKey)

			// Check if node has already voted
			if voted[m.Step] == nil {
				voted[m.Step] = make(map[string]bool)
			}

			if voted[m.Step][pkEd] {
				break
			}

			// Log vote
			voted[m.Step][pkEd] = true

			// Store set if it's ours
			if bytes.Equal(m.PubKey, ctx.Keys.EdPubKeyBytes()) {
				sets[m.Step] = pl.VoteSet
			}

			// Get amount of votes
			committee, err := sortition.CreateCommittee(m.Round, ctx.W, m.Step,
				ctx.Committee, ctx.NodeWeights)
			if err != nil {
				// Log
				c <- false
				return
			}

			votes := sortition.Verify(committee, m.PubKey)
			counter[m.Step] += int(votes)

			// Gossip the message
			if err := ctx.SendMessage(ctx.Magic, m); err != nil {
				// Log
				c <- false
				return
			}

			// Check if we have exceeded the limit
			size := len(ctx.Committee)
			if size > 50 {
				size = 50
			}

			limit := float64(size) * 0.75
			if counter[m.Step] < int(limit) {
				break
			}

			// // Populate certificate
			// ctx.Certificate.SRPubKeys = make([][]byte, len(pl.VoteSet))
			// for i := 0; i < len(pl.VoteSet); i++ {
			// 	pkBLS := hex.EncodeToString(pl.VoteSet[i].PubKey)
			// 	ctx.Certificate.SRPubKeys = append(ctx.Certificate.SRPubKeys,
			//		ctx.NodeBLS[pkBLS])
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
			ctx.WinningSigSetHash = pl.SetHash
			c <- true
			ctx.QuitChan <- true
			return
		}
	}
}

// SendSigSet will send out a signature set agreement message.
func SendSigSet(ctx *user.Context) error {
	// Create payload, signature and message
	pl, err := consensusmsg.NewSigSetAgreement(ctx.WinningBlockHash, ctx.SigSetHash,
		ctx.SigSetVotes, atomic.LoadUint32(&ctx.SigSetStep))
	if err != nil {
		return err
	}

	sigEd, err := ctx.CreateSignature(pl, atomic.LoadUint32(&ctx.SigSetStep))
	if err != nil {
		return err
	}

	msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
		atomic.LoadUint32(&ctx.SigSetStep), sigEd, ctx.Keys.EdPubKeyBytes(), pl)
	if err != nil {
		return err
	}

	// Send it to our own agreement channel
	ctx.SigSetAgreementChan <- msg
	return nil
}
