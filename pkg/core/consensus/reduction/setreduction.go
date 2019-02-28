package reduction

import (
	"bytes"
	"encoding/hex"
	"sync/atomic"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/sortition"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

// SignatureSet is the signature set reduction phase of the consensus.
func SignatureSet(ctx *user.Context) error {
	// Set up a fallback value
	fallback := make([]byte, 32)

	// Clear our votes out, so that we get a fresh set for this phase
	ctx.SigSetVotes = make([]*consensusmsg.Vote, 0)

	// Vote on collected signature set
	if err := sigSetVote(ctx); err != nil {
		return err
	}

	// Receive all other votes
	if err := countSigSetVotes(ctx); err != nil {
		return err
	}

	atomic.AddUint32(&ctx.SigSetStep, 1)

	// Vote on collected signature set
	if err := sigSetVote(ctx); err != nil {
		return err
	}

	// Receive all other votes
	if err := countSigSetVotes(ctx); err != nil {
		return err
	}

	// If SigSetHash is fallback, the committee has agreed to exit and restart
	// consensus.
	if !bytes.Equal(ctx.SigSetHash, fallback) {
		if err := agreement.SendSigSet(ctx); err != nil {
			return err
		}
	}

	atomic.AddUint32(&ctx.SigSetStep, 1)
	return nil
}

func sigSetVote(ctx *user.Context) error {
	select {
	case <-ctx.QuitChan:
		// Send another value to get through the phase
		ctx.QuitChan <- true
		return nil
	default:
		// Set committee first
		if err := ctx.SetCommittee(atomic.LoadUint32(&ctx.SigSetStep)); err != nil {
			return err
		}

		// If we are not in the committee, then don't vote
		votes := sortition.Verify(ctx.CurrentCommittee, []byte(*ctx.Keys.EdPubKey))
		if votes == 0 {
			return nil
		}

		// Sign signature set hash with BLS
		sigBLS, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, ctx.SigSetHash)
		if err != nil {
			return err
		}

		// Create signature set vote message to gossip
		pl, err := consensusmsg.NewSigSetReduction(ctx.WinningBlockHash, ctx.SigSetHash, sigBLS,
			ctx.Keys.BLSPubKey.Marshal())
		if err != nil {
			return err
		}

		sigEd, err := ctx.CreateSignature(pl, atomic.LoadUint32(&ctx.SigSetStep))
		if err != nil {
			return err
		}

		msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
			atomic.LoadUint32(&ctx.SigSetStep), sigEd, []byte(*ctx.Keys.EdPubKey), pl)
		if err != nil {
			return err
		}

		ctx.SigSetReductionChan <- msg
		return nil
	}
}

func countSigSetVotes(ctx *user.Context) error {
	// Set vote limit
	voteLimit := int(0.75 * float64(len(ctx.Committee)))
	if voteLimit > 50 {
		voteLimit = 50
	}

	limit := len(ctx.Committee)
	if limit > 50 {
		limit = 50
	}

	var receivedCount int

	// Keep a counter of how many votes have been cast for a specific set
	counts := make(map[string]uint8)

	// Keep track of all nodes who have voted
	voters := make(map[string]bool)

	// Start the timer
	timer := time.NewTimer(user.StepTime * (time.Duration(ctx.Multiplier)))

	// Empty queue
	prErr := msg.ProcessSigSetQueue(ctx)
	if prErr != nil && prErr.Priority == prerror.High {
		return prErr.Err
	}

	for {
		// If we got all votes for this current phase, we move on.
		if receivedCount == limit {
			ctx.SigSetHash = make([]byte, 32)
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

			ctx.SigSetHash = make([]byte, 32)
			return nil
		case m := <-ctx.SigSetReductionChan:
			if m.Round != ctx.Round || m.Step != atomic.LoadUint32(&ctx.SigSetStep) {
				break
			}

			pl := m.Payload.(*consensusmsg.SigSetReduction)
			pkEd := hex.EncodeToString(m.PubKey)

			// Check if this node's vote is already recorded
			if voters[pkEd] {
				break
			}

			// Get amount of votes
			votes := sortition.Verify(ctx.CurrentCommittee, m.PubKey)

			// Log information
			voters[pkEd] = true
			setStr := hex.EncodeToString(pl.SigSetHash)
			counts[setStr] += votes
			receivedCount += int(votes)
			sigSetVote, err := consensusmsg.NewVote(pl.SigSetHash, pl.PubKeyBLS,
				pl.SigBLS, atomic.LoadUint32(&ctx.SigSetStep))
			if err != nil {
				return err
			}

			for i := uint8(0); i < votes; i++ {
				ctx.SigSetVotes = append(ctx.SigSetVotes, sigSetVote)
			}

			// Gossip the message
			if err := ctx.SendMessage(ctx.Magic, m); err != nil {
				return err
			}

			// If a set exceeds vote threshold, we will end the loop.
			if counts[setStr] < uint8(voteLimit) {
				break
			}

			timer.Stop()

			// Set signature set hash
			ctx.SigSetHash = pl.SigSetHash

			// We will also cut all the votes that did not vote for the winning block.
			for i, vote := range ctx.SigSetVotes {
				if !bytes.Equal(vote.Hash, ctx.SigSetHash) {
					ctx.SigSetVotes = append(ctx.SigSetVotes[:i], ctx.SigSetVotes[i+1:]...)
				}
			}

			return nil
		}
	}
}
