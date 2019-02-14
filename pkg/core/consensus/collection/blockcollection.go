package collection

import (
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// Block is the function that is ran by provisioners and generators during block generation.
// After the function terminates, the context object should hold a block hash that
// came with the highest score found during message processing, which should then
// be voted on in later phases, if the corresponding block was also received for it.
func Block(ctx *user.Context) error {
	// Keeps track of those who have already propagated their score messages
	senders := make(map[string]bool)

	// Keep track of the highest bid score seen
	var highest uint64

	// Empty out our CandidateBlocks map
	ctx.CandidateBlocks = make(map[string]*block.Block)

	timer := time.NewTimer(user.CandidateTime * (time.Duration(ctx.Multiplier)))

	for {
		select {
		case <-timer.C:
			return nil
		case m := <-ctx.CandidateChan:
			pl := m.Payload.(*consensusmsg.Candidate)
			blockHash := hex.EncodeToString(pl.Block.Header.Hash)

			// Add candidate block to map of known candidate blocks, if unknown
			if ctx.CandidateBlocks[blockHash] != nil {
				break
			}
			ctx.CandidateBlocks[blockHash] = pl.Block

		case m := <-ctx.CandidateScoreChan:
			pl := m.Payload.(*consensusmsg.CandidateScore)
			pkEd := hex.EncodeToString(m.PubKey)

			// Add candidate score to map of known candidate scores, if unknown
			if senders[pkEd] {
				break
			}
			senders[pkEd] = true

			// If the received score is higher than our current one, replace
			// the current blockhash and score with the received block hash and score.
			if pl.Score > highest {
				highest = pl.Score
				ctx.BlockHash = pl.CandidateHash
			}

		}
	}
}
