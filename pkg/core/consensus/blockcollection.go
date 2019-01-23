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
	var blocks []*block.Block
	var highest uint64
	timer := time.NewTimer(candidateTime)

	for {
	out:
		select {
		case <-timer.C:
			for _, block := range blocks {
				if bytes.Equal(block.Header.Hash, ctx.BlockHash) {
					ctx.CandidateBlock = block
				}
			}

			return nil
		case m := <-ctx.msgs:
			// Type checks
			if m.ID == consensusmsg.CandidateID {
				pl := m.Payload.(*consensusmsg.Candidate)
				blocks = append(blocks, pl.Block)
			}

			if m.ID == consensusmsg.CandidateScoreID {
				pl := m.Payload.(*consensusmsg.CandidateScore)

				// Check if this node's candidate was already recorded
				for _, sender := range senders {
					if bytes.Equal(sender, m.PubKey) {
						break out
					}
				}

				// Verify the message
				valid, score, err := processMsg(ctx, m)
				if err != nil {
					return err
				}

				if !valid {
					break
				}

				// If the score is higher than our current one, replace
				if score > highest {
					highest = score
					ctx.BlockHash = pl.CandidateHash
				}
			}
		}
	}
}
