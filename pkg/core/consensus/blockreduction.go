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

	// Create fallback value (zero)
	fallback := make([]byte, 32)

	// Save starting value
	var startHash []byte
	startHash = append(startHash, ctx.BlockHash...)

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
	// So we will vote again for the first value.
	if ctx.BlockHash == nil {
		ctx.BlockHash = startHash
	}

	if err := committeeVoteReduction(ctx); err != nil {
		return err
	}

	if err := countVotesReduction(ctx); err != nil {
		return err
	}

	ctx.Step++

	// If BlockHash is nil, no clear winner was found within the time limit.
	// So we will return a fallback value instead.
	if ctx.BlockHash == nil {
		ctx.BlockHash = fallback
		ctx.BlockVotes = nil
		return nil
	}

	// If we did get a result at the end of the phases, send block set agreement message
	if err := sendSetAgreement(ctx); err != nil {
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
		blsPubBytes := ctx.Keys.BLSPubKey.Marshal()
		pl, err := consensusmsg.NewReduction(ctx.Score, ctx.BlockHash, sigBLS, blsPubBytes)
		if err != nil {
			return err
		}

		sigEd, err := createSignature(ctx, pl)
		if err != nil {
			return err
		}

		msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash, ctx.Step, sigEd,
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
		case m := <-ctx.ReductionChan:
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

			// If votes is zero, then the reduction message was most likely
			// faulty, so we will ignore it.
			if votes == 0 || !valid {
				break
			}

			// Log new information
			voters = append(voters, m.PubKey)
			hashStr := hex.EncodeToString(pl.BlockHash)
			counts[hashStr] += votes
			blockVote, err := consensusmsg.NewVote(pl.BlockHash, pl.PubKeyBLS, pl.SigBLS, ctx.Step)
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
