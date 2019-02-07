package collection

import (
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// Block is the function that is ran by provisioners and generators during block generation.
// After the function terminates, the context object should hold a block hash that
// came with the highest score it found during message processing, which should then
// be voted on in later phases.
func Block(ctx *user.Context) error {
	// Keep track of those who have already propagated their messages
	senders := make(map[string]bool)

	// Keep track of the highest bid score seen
	var highest uint64

	// Empty out our CandidateBlocks map
	ctx.CandidateBlocks = make(map[string]*block.Block)

	// Start the timer
	timer := time.NewTimer(user.CandidateTime * (time.Duration(ctx.Multiplier) * time.Second))

	for {
		select {
		case <-timer.C:
			return nil
		case m := <-ctx.CandidateChan:
			pl := m.Payload.(*consensusmsg.Candidate)
			blockHash := hex.EncodeToString(pl.Block.Header.Hash)

			// See if we already have it
			if ctx.CandidateBlocks[blockHash] != nil {
				break
			}

			// Verify the message
			_, err := msg.Process(ctx, m)
			if err != nil {
				if err.Priority == prerror.High {
					return err
				}

				// Discard if invalid
				break
			}

			// Add to the mapping
			ctx.CandidateBlocks[blockHash] = pl.Block
		case m := <-ctx.CandidateScoreChan:
			pl := m.Payload.(*consensusmsg.CandidateScore)
			pkEd := hex.EncodeToString(m.PubKey)

			// Check if this node's candidate was already recorded
			if senders[pkEd] {
				break
			}

			// Verify the message
			_, err := msg.Process(ctx, m)
			if err != nil {
				if err.Priority == prerror.High {
					return err
				}

				// Discard if invalid
				break
			}

			// Log information
			senders[pkEd] = true

			// If the score is higher than our current one, replace
			if pl.Score > highest {
				highest = pl.Score
				ctx.BlockHash = pl.CandidateHash
			}
		}
	}
}
