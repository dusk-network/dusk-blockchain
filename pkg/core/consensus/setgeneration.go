package consensus

import (
	"bytes"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// SignatureSetGeneration will generate a signature set message, gossip it, and
// then collect all other messages, then retaining the most voted set for the
// signature set reduction phase.
func SignatureSetGeneration(ctx *Context) error {
	// Create our own signature set candidate message
	pl, err := consensusmsg.NewSigSetCandidate(ctx.BlockHash, ctx.SigSetVotes,
		ctx.Keys.BLSPubKey.Marshal(), ctx.Score)
	if err != nil {
		return err
	}

	sigEd, err := createSignature(ctx, pl)
	if err != nil {
		return err
	}

	msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
		ctx.Step, sigEd, []byte(*ctx.Keys.EdPubKey), pl)
	if err != nil {
		return err
	}

	// Gossip msg
	if err := ctx.SendMessage(ctx.Magic, msg); err != nil {
		return err
	}

	// Collect signature set with highest score, and set our context value
	// to the winner.

	// Keep track of those who have voted
	voters := make(map[string]bool)
	pk := hex.EncodeToString([]byte(*ctx.Keys.EdPubKey))

	// Log our own key
	voters[pk] = true

	// Initialize container for all vote sets, and add our own
	ctx.AllVotes = make(map[string][]*consensusmsg.Vote)
	sigSetHash, err := hashSigSetVotes(ctx)
	if err != nil {
		return err
	}

	ctx.AllVotes[hex.EncodeToString(sigSetHash)] = ctx.SigSetVotes
	highest := ctx.weight

	// Start timer
	timer := time.NewTimer(stepTime)

	for {
	out:
		select {
		case <-timer.C:
			return nil
		case m := <-ctx.SigSetCandidateChan:
			pl := m.Payload.(*consensusmsg.SigSetCandidate)
			pkEd := hex.EncodeToString(m.PubKey)

			// Check if this node's signature set is already recorded
			if voters[pkEd] {
				break out
			}

			// Verify the message
			valid, stake, err := processMsg(ctx, m)
			if err != nil {
				return err
			}

			// Discard if it's invalid
			if !valid {
				break
			}

			// Log information
			voters[pkEd] = true

			// If the stake is higher than our current one, replace
			if stake > highest {
				highest = stake
				ctx.SigSetVotes = pl.SignatureSet
			}
		}
	}
}

// Returns the hash of ctx.SigSetVotes
func hashSigSetVotes(ctx *Context) ([]byte, error) {
	// Encode signature set
	buf := new(bytes.Buffer)
	for _, vote := range ctx.SigSetVotes {
		if err := vote.Encode(buf); err != nil {
			return nil, err
		}
	}

	// Hash bytes and set it on context
	sigSetHash, err := hash.Sha3256(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return sigSetHash, nil
}
