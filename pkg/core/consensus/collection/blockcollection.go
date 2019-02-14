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
	// Use this to store block candidate hashes
	var received [][]byte

	// Keep track of the highest bid score seen
	var highest uint64

	// Empty out our CandidateBlocks map
	ctx.CandidateBlocks = make(map[string]*block.Block)

	timer := time.NewTimer(user.CandidateTime * (time.Duration(ctx.Multiplier)))

	for {
		select {
		case <-timer.C:
			// Walk backwards through the received slice (highest score first)
			// to find the first hash that we received a block for. This will
			// be the block we will vote on.
			for i := len(received) - 1; i >= 0; i-- {
				blockHash := hex.EncodeToString(received[i])
				if ctx.CandidateBlocks[blockHash] != nil {
					ctx.BlockHash = received[i]
					break
				}
			}

			return nil
		case m := <-ctx.CandidateChan:
			pl := m.Payload.(*consensusmsg.Candidate)
			blockHash := hex.EncodeToString(pl.Block.Header.Hash)

			// Add candidate block to map of known candidate blocks, if unknown
			if ctx.CandidateBlocks[blockHash] != nil {
				break
			}
			ctx.CandidateBlocks[blockHash] = pl.Block

			// Gossip it to the rest of the network
			ctx.SendMessage(ctx.Magic, m)
		case m := <-ctx.CandidateScoreChan:
			pl := m.Payload.(*consensusmsg.CandidateScore)

			// If the received score is higher than our current one, replace
			// the current score with the received score, and store the block hash.
			if pl.Score > highest {
				highest = pl.Score
				received = append(received, pl.CandidateHash)

				// Gossip it to the rest of the network
				ctx.SendMessage(ctx.Magic, m)
			}

		}
	}
}
