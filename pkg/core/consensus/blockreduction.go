package consensus

import (
	"bytes"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// BlockReduction is the main function that runs during block reduction phase.
func BlockReduction(ctx *Context) error {
	// First, clear our votes out, so that we get a fresh set for this phase.
	ctx.BlockVotes = make([]*consensusmsg.Vote, 0)

	// Vote on passed block
	if err := committeeVoteReduction(ctx); err != nil {
		return err
	}

	// Receive all other votes
	if err := countVotesReduction(ctx); err != nil {
		return err
	}

	ctx.Step++

	// If BlockHash is nil, no clear winner was found within the time limit.
	// So we will exit and restart the consensus.
	if ctx.BlockHash == nil {
		return nil
	}

	if err := committeeVoteReduction(ctx); err != nil {
		return err
	}

	if err := countVotesReduction(ctx); err != nil {
		return err
	}

	ctx.Step++

	// If BlockHash is nil, no clear winner was found within the time limit.
	// So we will exit and restart the consensus.
	if ctx.BlockHash == nil {
		return nil
	}

	// If we did get a result, send block set agreement message
	if err := sendSetAgreement(ctx, ctx.BlockVotes); err != nil {
		return err
	}

	return nil
}

func committeeVoteReduction(ctx *Context) error {
	// Run sortition
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.Step,
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
		pl, err := consensusmsg.NewReduction(ctx.Score, ctx.BlockHash, sigBLS, ctx.Keys.BLSPubKey.Marshal())
		if err != nil {
			return err
		}

		// Sign the payload
		sigEd, err := createSignature(ctx, pl)
		if err != nil {
			return err
		}

		// Create message
		msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash, ctx.Step, sigEd,
			[]byte(*ctx.Keys.EdPubKey), pl)
		if err != nil {
			return err
		}

		// Gossip message
		if err := ctx.SendMessage(ctx.Magic, msg); err != nil {
			return err
		}
	}

	return nil
}

func countVotesReduction(ctx *Context) error {
	// Keep a counter of how many votes have been cast for a specific block
	counts := make(map[string]uint64)

	// Keep track of all nodes who have voted
	voters := make(map[string]bool)

	// Add our own information beforehand
	voters[hex.EncodeToString([]byte(*ctx.Keys.EdPubKey))] = true
	counts[hex.EncodeToString(ctx.BlockHash)] += ctx.votes

	// Start the timer
	timer := time.NewTimer(stepTime)

	for {
	out:
		select {
		case <-timer.C:
			ctx.BlockHash = nil
			return nil
		case m := <-ctx.ReductionChan:
			pl := m.Payload.(*consensusmsg.Reduction)
			pkEd := hex.EncodeToString(m.PubKey)

			// Check if this node's vote is already recorded
			if voters[pkEd] {
				break out
			}

			// Verify the message score and get back it's contents
			valid, votes, err := processMsg(ctx, m)
			if err != nil {
				return err
			}

			// If votes is zero, then the reduction message was most likely
			// faulty, so we will ignore it.
			if votes == 0 || !valid {
				break
			}

			// Log new information
			voters[pkEd] = true
			hashStr := hex.EncodeToString(pl.BlockHash)
			counts[hashStr] += votes
			blockVote, err := consensusmsg.NewVote(pl.BlockHash, pl.PubKeyBLS, pl.SigBLS, pl.Score, ctx.Step)
			if err != nil {
				return err
			}

			ctx.BlockVotes = append(ctx.BlockVotes, blockVote)

			// If a block exceeds the vote threshold, we will end the loop.
			if counts[hashStr] >= ctx.VoteLimit {
				timer.Stop()
				ctx.BlockHash = pl.BlockHash

				// We will also cut all the votes that did not vote for the winning block.
				for i, vote := range ctx.BlockVotes {
					if !bytes.Equal(vote.Hash, ctx.BlockHash) {
						ctx.BlockVotes = append(ctx.BlockVotes[:i], ctx.BlockVotes[i+1:]...)
					}
				}

				return nil
			}
		}
	}
}
