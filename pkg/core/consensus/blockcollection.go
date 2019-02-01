package consensus

import (
	"bytes"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// BlockCollection is the function that is ran by provisioners during block generation.
// After the function terminates, the context object should hold a block hash that
// came with the highest score it found during message processing, which should then
// be voted on in later phases.
func BlockCollection(ctx *Context) error {
	var senders [][]byte
	blocks := make([]*block.Block, 0)
	var highest uint64
	timer := time.NewTimer(candidateTime)

	for {
	out:
		select {
		case <-timer.C:
			// Set CandidateBlock on context, if we received it
			for _, block := range blocks {
				if bytes.Equal(block.Header.Hash, ctx.BlockHash) {
					ctx.CandidateBlock = block
				}
			}

			return nil
		case m := <-ctx.CandidateChan:
			pl := m.Payload.(*consensusmsg.Candidate)

			// See if we already have it
			for _, block := range blocks {
				if bytes.Equal(pl.Block.Header.Hash, block.Header.Hash) {
					break out
				}
			}

			// Verify the message
			valid, _, err := processMsg(ctx, m)
			if err != nil {
				return err
			}

			// Discard if invalid
			if !valid {
				break
			}

			blocks = append(blocks, pl.Block)
		case m := <-ctx.CandidateScoreChan:
			pl := m.Payload.(*consensusmsg.CandidateScore)

			// Check if this node's candidate was already recorded
			for _, sender := range senders {
				if bytes.Equal(sender, m.PubKey) {
					break out
				}
			}

			// Verify the message
			valid, _, err := processMsg(ctx, m)
			if err != nil {
				return err
			}

			// Discard if invalid
			if !valid {
				break
			}

			// If the score is higher than our current one, replace
			if pl.Score > highest {
				highest = pl.Score
				ctx.BlockHash = pl.CandidateHash
			}
		}
	}
}
