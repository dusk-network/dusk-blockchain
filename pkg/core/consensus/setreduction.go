package consensus

import (
	"bytes"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

// SignatureSetReduction is the signature set reduction phase of the consensus.
func SignatureSetReduction(ctx *Context) error {
	// Set up a fallback value
	fallback := make([]byte, 32)

	// Vote on collected signature set
	if err := committeeVoteSigSet(ctx); err != nil {
		return err
	}

	// Receive all other votes
	if err := countVotesSigSet(ctx); err != nil {
		return err
	}

	ctx.Step++

	// If we timed out, exit the loop and go back to signature set generation
	if ctx.SigSetHash == nil {
		ctx.SigSetHash = fallback
	}

	if err := committeeVoteSigSet(ctx); err != nil {
		return err
	}

	if err := countVotesSigSet(ctx); err != nil {
		return err
	}

	ctx.Step++

	// If we timed out, exit the loop and go back to signature set generation
	if ctx.SigSetHash == nil {
		return nil
	}

	// If SigSetHash is fallback, the committee has agreed to exit and restart
	// consensus.
	if bytes.Equal(ctx.SigSetHash, fallback) {
		ctx.SigSetHash = nil
		return nil
	}

	// If we got a result, populate certificate, send message to
	// set agreement and terminate
	if err := SendSetAgreement(ctx, ctx.SigSetVotes); err != nil {
		return err
	}

	return nil
}

func committeeVoteSigSet(ctx *Context) error {
	sigSetHash, err := hashSigSetVotes(ctx.SigSetVotes)
	if err != nil {
		return err
	}

	ctx.SigSetHash = sigSetHash
	setStr := hex.EncodeToString(sigSetHash)
	ctx.AllVotes[setStr] = ctx.SigSetVotes

	// Sign signature set hash with BLS
	sigBLS, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, ctx.SigSetHash)
	if err != nil {
		return err
	}

	// Create signature set vote message to gossip
	pl, err := consensusmsg.NewSigSetVote(ctx.BlockHash, ctx.SigSetHash, sigBLS,
		ctx.Keys.BLSPubKey.Marshal(), ctx.Score)
	if err != nil {
		return err
	}

	sigEd, err := CreateSignature(ctx, pl)
	if err != nil {
		return err
	}

	msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash, ctx.Step,
		sigEd, []byte(*ctx.Keys.EdPubKey), pl)
	if err != nil {
		return err
	}

	// Gossip message
	if err := ctx.SendMessage(ctx.Magic, msg); err != nil {
		return err
	}

	return nil
}

func countVotesSigSet(ctx *Context) error {
	// Keep a counter of how many votes have been cast for a specific block
	counts := make(map[string]uint64)

	// Keep track of all nodes who have voted
	voters := make(map[string]bool)

	// Add our own information beforehand
	voters[hex.EncodeToString([]byte(*ctx.Keys.EdPubKey))] = true
	counts[hex.EncodeToString(ctx.SigSetHash)] += ctx.Weight

	// Start the timer
	timer := time.NewTimer(StepTime * (time.Duration(ctx.Multiplier) * time.Second))

	for {
		select {
		case <-timer.C:
			ctx.SigSetHash = nil
			return nil
		case m := <-ctx.SigSetVoteChan:
			pl := m.Payload.(*consensusmsg.SigSetVote)
			pkEd := hex.EncodeToString(m.PubKey)

			// Check if this node's vote is already recorded
			if voters[pkEd] {
				break
			}

			// Verify the message
			stake, err := ProcessMsg(ctx, m)
			if err != nil {
				if err.Priority == prerror.High {
					return err.Err
				}

				// Discard if invalid
				break
			}

			// If sender hasn't staked, discard
			if stake == 0 {
				break
			}

			// Log information
			voters[pkEd] = true
			setStr := hex.EncodeToString(pl.SigSetHash)
			counts[setStr] += stake

			// If a set exceeds vote threshold, we will end the loop.
			if counts[setStr] < ctx.VoteLimit {
				break
			}

			timer.Stop()

			// Set signature set hash
			ctx.SigSetHash = pl.SigSetHash

			// Set vote set to winning hash
			ctx.SigSetVotes = ctx.AllVotes[setStr]
			return nil
		}
	}
}
