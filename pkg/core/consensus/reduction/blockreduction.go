package reduction

import (
	"bytes"
	"encoding/hex"
	"sync/atomic"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/sortition"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// Block is the main function that runs during block reduction phase.
func Block(ctx *user.Context) error {
	// Set up a fallback value
	fallback := make([]byte, 32)

	// If we made it here without getting a block, set blockhash to fallback value.
	if ctx.BlockHash == nil {
		ctx.BlockHash = fallback
	}

	// Clear our votes out, so that we get a fresh set for this phase.
	ctx.BlockVotes = make([]*consensusmsg.Vote, 0)

	// Vote on passed block
	if err := blockVote(ctx); err != nil {
		return err
	}

	// Receive all other votes
	if err := countBlockVotes(ctx); err != nil {
		return err
	}

	atomic.AddUint32(&ctx.BlockStep, 1)

	// Vote on passed block
	if err := blockVote(ctx); err != nil {
		return err
	}

	// Receive all other votes
	if err := countBlockVotes(ctx); err != nil {
		return err
	}

	// If BlockHash is fallback, the committee has agreed to exit and restart
	// consensus.
	if !bytes.Equal(ctx.BlockHash, fallback) {
		if err := agreement.SendBlock(ctx); err != nil {
			return err
		}
	}

	atomic.AddUint32(&ctx.BlockStep, 1)
	return nil
}

func blockVote(ctx *user.Context) error {
	select {
	case <-ctx.QuitChan:
		// Send another value to get through the phase
		ctx.QuitChan <- true
		return nil
	default:
		// Set committee first
		if err := sortition.SetCommittee(ctx, atomic.LoadUint32(&ctx.BlockStep)); err != nil {
			return err
		}

		// If we are not in the committee, then don't vote
		pkEd := hex.EncodeToString(ctx.Keys.EdPubKeyBytes())
		if ctx.CurrentCommittee[pkEd] == 0 {
			return nil
		}

		// Sign block hash with BLS
		sigBLS, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, ctx.BlockHash)
		if err != nil {
			return err
		}

		// Create reduction payload to gossip
		pl, err := consensusmsg.NewBlockReduction(ctx.BlockHash, sigBLS, ctx.Keys.BLSPubKey.Marshal())
		if err != nil {
			return err
		}

		// Sign the payload
		sigEd, err := ctx.CreateSignature(pl, atomic.LoadUint32(&ctx.BlockStep))
		if err != nil {
			return err
		}

		// Create message
		msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
			atomic.LoadUint32(&ctx.BlockStep), sigEd, ctx.Keys.EdPubKeyBytes(), pl)
		if err != nil {
			return err
		}

		ctx.BlockReductionChan <- msg
		return nil
	}
}

func countBlockVotes(ctx *user.Context) error {
	// Set vote limit
	voteLimit := int(0.75 * float64(len(*ctx.Committee)))
	if voteLimit > 38 {
		voteLimit = 38
	}

	limit := len(*ctx.Committee)
	if limit > 50 {
		limit = 50
	}

	var receivedCount int

	// Keep a counter of how many votes have been cast for a specific block
	counts := make(map[string]uint8)

	// Keep track of all nodes who have voted
	voters := make(map[string]bool)

	// Start the timer
	timer := time.NewTimer(user.StepTime * (time.Duration(ctx.Multiplier)))

	// Empty queue
	prErr := msg.ProcessBlockQueue(ctx)
	if prErr != nil && prErr.Priority == prerror.High {
		return prErr.Err
	}

	for {
		// If we got all votes for this current phase, we move on.
		if receivedCount == limit {
			ctx.BlockHash = make([]byte, 32)
			return nil
		}

		select {
		case <-ctx.QuitChan:
			// Send another value to get through the phase
			ctx.QuitChan <- true
			return nil
		case <-timer.C:
			// Increase multiplier
			if ctx.Multiplier < 10 {
				ctx.Multiplier = ctx.Multiplier * 2
			}

			ctx.BlockHash = make([]byte, 32)
			return nil
		case m := <-ctx.BlockReductionChan:
			if m.Round != ctx.Round || m.Step != atomic.LoadUint32(&ctx.BlockStep) {
				break
			}

			pl := m.Payload.(*consensusmsg.BlockReduction)
			pkEd := hex.EncodeToString(m.PubKey)

			// Check if this node's vote is already recorded
			if voters[pkEd] {
				break
			}

			// Get amount of votes
			votes := ctx.CurrentCommittee[pkEd]

			// Log new information
			voters[pkEd] = true
			hashStr := hex.EncodeToString(pl.BlockHash)
			counts[hashStr] += votes
			receivedCount += int(votes)
			blockVote, err := consensusmsg.NewVote(pl.BlockHash, pl.PubKeyBLS,
				pl.SigBLS, atomic.LoadUint32(&ctx.BlockStep))
			if err != nil {
				return err
			}

			for i := uint8(0); i < votes; i++ {
				ctx.BlockVotes = append(ctx.BlockVotes, blockVote)
			}

			// Gossip the message
			if err := ctx.SendMessage(ctx.Magic, m); err != nil {
				return err
			}

			// If a block doesnt exceed the vote threshold, we keep going.
			if counts[hashStr] < uint8(voteLimit) {
				break
			}

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
