package collection

import (
	"bytes"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// Block is the function that is ran by provisioners and generators during block generation.
// After the function terminates, the context object should hold a block hash that
// came with the highest score found during message processing, which should then
// be voted on in later phases, if the corresponding block was also received for it.
func Block(ctx *user.Context) error {
	// Keep track of the highest bid score seen
	var highest uint64
	timer := time.NewTimer(user.CandidateTime * (time.Duration(ctx.Multiplier)))

	// Process queue
	if err := msg.ProcessQueue(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-timer.C:
			return nil
		case m := <-ctx.CandidateScoreChan:
			pl := m.Payload.(*consensusmsg.CandidateScore)

			// If the received score is higher than our current one, replace
			// the current score with the received score, and store the block hash.
			if pl.Score > highest {
				highest = pl.Score
				ctx.BlockHash = pl.CandidateHash

				// Gossip it to the rest of the network
				ctx.SendMessage(ctx.Magic, m)
			}
		case m := <-ctx.CandidateChan:
			pl := m.Payload.(*consensusmsg.Candidate)

			// Add candidate block to map of known candidate blocks, if unknown
			if !bytes.Equal(pl.Block.Header.Hash, ctx.BlockHash) {
				break
			}

			// If we already received the block before, we avoid gossiping it again.
			if ctx.CandidateBlock.Header.Hash != nil &&
				bytes.Equal(ctx.CandidateBlock.Header.Hash, pl.Block.Header.Hash) {
				break
			}

			ctx.CandidateBlock = pl.Block

			// Gossip it to the rest of the network
			ctx.SendMessage(ctx.Magic, m)
		}
	}
}
