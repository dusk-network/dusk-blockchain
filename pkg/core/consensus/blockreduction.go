package consensus

import (
	"bytes"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

type role struct {
	part  string
	round uint64
	step  uint8
}

// BlockReduction is the main function that runs during block reduction phase.
// Once the reduction phase is finished, the context object holds a block hash
// that will then be voted on in the binary agreement phase.
func BlockReduction(ctx *Context) error {
	// Step 1
	ctx.step = 1

	// Prepare empty block
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		return err
	}

	// If no candidate block was found, then we use the empty block
	if ctx.BlockHash == nil {
		ctx.BlockHash = emptyBlock.Header.Hash
		ctx.Empty = true
	}

	// Vote on passed block
	if err := committeeVoteReduction(ctx); err != nil {
		return err
	}

	// Receive all other votes
	if err := countVotesReduction(ctx); err != nil {
		return err
	}

	// Step 2
	ctx.step++

	// If retHash is nil, no clear winner was found within the time limit.
	// So we will vote on an empty block instead.
	if ctx.BlockHash == nil {
		ctx.BlockHash = emptyBlock.Header.Hash
		ctx.Empty = true
	}

	if err := committeeVoteReduction(ctx); err != nil {
		return err
	}

	if err := countVotesReduction(ctx); err != nil {
		return err
	}

	// If retHash is nil, no clear winner was found within the time limit.
	// So we will return an empty block instead.
	if ctx.BlockHash == nil {
		ctx.BlockHash = emptyBlock.Header.Hash
		ctx.Empty = true
	}

	return nil
}

func committeeVoteReduction(ctx *Context) error {
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.step,
	}

	if err := sortition(ctx, role); err != nil {
		return err
	}

	if ctx.votes > 0 {
		// Sign block hash with BLS
		sigBLS, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, ctx.BlockHash)
		if err != nil {
			return err
		}

		// Create reduction payload to gossip
		blsPubBytes := ctx.Keys.BLSPubKey.Marshal()
		pl, err := consensusmsg.NewReduction(ctx.Score, ctx.step, ctx.BlockHash, sigBLS, blsPubBytes)
		if err != nil {
			return err
		}

		sigEd, err := createSignature(ctx, pl)
		msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash, sigEd,
			[]byte(*ctx.Keys.EdPubKey), pl)
		if err != nil {
			return err
		}

		if err := ctx.SendMessage(ctx.Magic, msg); err != nil {
			return err
		}
	}

	return nil
}

func countVotesReduction(ctx *Context) error {
	counts := make(map[string]uint64)
	var voters [][]byte
	voters = append(voters, []byte(*ctx.Keys.EdPubKey))
	counts[hex.EncodeToString(ctx.BlockHash)] += ctx.votes
	timer := time.NewTimer(stepTime)

	for {
	out:
		select {
		case <-timer.C:
			ctx.BlockHash = nil
			return nil
		case m := <-ctx.msgs:
			// Check first off if this message is the right one, if not
			// we discard it.
			if m.ID != consensusmsg.ReductionID {
				break
			}

			pl := m.Payload.(*consensusmsg.Reduction)

			// Check if this node's vote is already recorded
			for _, voter := range voters {
				if bytes.Equal(voter, m.PubKey) {
					break out
				}
			}

			// Verify the message score and get back it's contents
			valid, votes, err := processMsg(ctx, m)
			if err != nil {
				return err
			}

			if !valid {
				break
			}

			// If votes is zero, then the reduction message was most likely
			// faulty, so we will ignore it.
			if votes == 0 {
				break
			}

			// Log new information
			voters = append(voters, m.PubKey)
			hashStr := hex.EncodeToString(pl.BlockHash)
			counts[hashStr] += votes

			// If a block exceeds the vote threshold, we will end the loop.
			if counts[hashStr] > ctx.VoteLimit {
				timer.Stop()
				ctx.BlockHash = pl.BlockHash
				return nil
			}
		}
	}
}
