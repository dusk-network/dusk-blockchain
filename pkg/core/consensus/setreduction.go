package consensus

import (
	"bytes"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

// SignatureSetReduction is the signature set reduction phase of the consensus.
func SignatureSetReduction(ctx *Context) error {
	for ctx.Step = 1; ctx.Step < MaxSteps; ctx.Step++ {
		// Vote on collected signature set
		if err := committeeVoteSigSet(ctx); err != nil {
			return err
		}

		// Receive all other votes
		if err := countVotesSigSet(ctx); err != nil {
			return err
		}

		ctx.Step++

		// If we timed out, go back to the beginning of the loop
		if ctx.SigSetHash == nil {
			continue
		}

		if err := committeeVoteSigSet(ctx); err != nil {
			return err
		}

		if err := countVotesSigSet(ctx); err != nil {
			return err
		}

		// If we got a result, populate certificate
		if ctx.SigSetHash != nil {
			// And send message to set agreement and terminate
			if err := sendSetAgreement(ctx, ctx.SigSetVotes); err != nil {
				return err
			}

			return nil
		}
	}

	return nil
}

func committeeVoteSigSet(ctx *Context) error {
	// Encode signature set
	buf := new(bytes.Buffer)
	for _, vote := range ctx.SigSetVotes {
		if err := vote.Encode(buf); err != nil {
			return err
		}
	}

	// Hash bytes
	sigSetHash, err := hash.Sha3256(buf.Bytes())
	if err != nil {
		return err
	}

	ctx.SigSetHash = sigSetHash

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

	sigEd, err := createSignature(ctx, pl)
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
	counts := make(map[string]uint64)
	var voters [][]byte
	voters = append(voters, []byte(*ctx.Keys.EdPubKey))
	counts[hex.EncodeToString(ctx.SigSetHash)] += ctx.weight
	timer := time.NewTimer(stepTime)

	for {
	out:
		select {
		case <-timer.C:
			ctx.SigSetHash = nil
			return nil
		case m := <-ctx.SigSetVoteChan:
			pl := m.Payload.(*consensusmsg.SigSetVote)

			// Check if this node's vote is already recorded
			for _, voter := range voters {
				if bytes.Equal(voter, m.PubKey) {
					break out
				}
			}

			// Verify the message
			valid, stake, err := processMsg(ctx, m)
			if err != nil {
				return err
			}

			// Discard if invalid
			if stake == 0 || !valid {
				break
			}

			// Log information
			voters = append(voters, m.PubKey)
			setStr := hex.EncodeToString(pl.SigSetHash)
			counts[setStr] += stake

			// If a set exceeds vote threshold, we will end the loop.
			if counts[setStr] > ctx.VoteLimit {
				timer.Stop()
				ctx.SigSetHash = pl.SigSetHash
				return nil
			}
		}
	}
}
