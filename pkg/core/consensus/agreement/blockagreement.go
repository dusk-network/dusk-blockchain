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

// Block is the function that runs during the block reduction phase, used
// to collect vote sets and find a winning block. Block will run indefinitely
// until a decision is reached.
func Block(ctx *user.Context, c chan bool) {
	// Store our sets on every step for certificate generation
	sets := make(map[uint32][]*consensusmsg.Vote)

	// Make a counter to keep track of how many votes have been cast in a step
	counter := make(map[uint32]uint8)

	// Make a map to keep track if a node has voted in a certain step
	voted := make(map[uint32]map[string]bool)

	for {
		select {
		case m := <-ctx.BlockAgreementChan:
			if m.Round != ctx.Round {
				break
			}

			pl := m.Payload.(*consensusmsg.BlockAgreement)
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

			counter[m.Step] += committee[pkEd]

			// Gossip the message
			if err := ctx.SendMessage(ctx.Magic, m); err != nil {
				// Log
				c <- false
				return
			}

			// Check if we have exceeded the limit.
			size := len(*ctx.Committee)
			if size > 50 {
				size = 50
			}

			limit := float64(size) * 0.75
			if counter[m.Step] < uint8(limit) {
				break
			}

			// Set SigSetVotes for signature set phase
			ctx.WinningBlockVotes = pl.VoteSet
			if sets[m.Step] != nil {
				ctx.WinningBlockVotes = sets[m.Step]
			}

			ctx.WinningBlockHash = pl.BlockHash

			// We save the winning step for the signature set phase
			ctx.Certificate.BRStep = m.Step

			c <- true
			ctx.QuitChan <- true
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

	sigEd, err := ctx.CreateSignature(pl, atomic.LoadUint32(&ctx.BlockStep))
	if err != nil {
		return err
	}

	msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
		atomic.LoadUint32(&ctx.BlockStep), sigEd, ctx.Keys.EdPubKeyBytes(), pl)
	if err != nil {
		return err
	}

	// Send it to our own agreement channel
	ctx.BlockAgreementChan <- msg
	return nil
}
