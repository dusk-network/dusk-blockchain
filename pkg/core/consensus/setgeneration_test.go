package consensus

import (
	"bytes"
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

// This test will test both signature set generation and signature set collection.
func TestSignatureSetGeneration(t *testing.T) {
	// Put step timer down to avoid long waiting times
	stepTime = 3 * time.Second

	// Create dummy context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = hash
	votes, _, err := createVotesAndMsgs(ctx, 15)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetVotes = votes
	ctx.weight = 200
	ctx.VoteLimit = 20

	// Run sortition
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.Step,
	}

	if err := sortition(ctx, role); err != nil {
		t.Fatal(err)
	}

	// Shuffle vote set
	var otherVotes []*consensusmsg.Vote
	otherVotes = append(otherVotes, votes...)
	otherVotes[0] = otherVotes[1]

	// Create votes
	vote1, err := newSigSetCandidate(ctx, 500, otherVotes)
	if err != nil {
		t.Fatal(err)
	}

	vote2, err := newSigSetCandidate(ctx, 800, otherVotes)
	if err != nil {
		t.Fatal(err)
	}

	vote3, err := newSigSetCandidate(ctx, 1500, otherVotes)
	if err != nil {
		t.Fatal(err)
	}

	allVotes := []*payload.MsgConsensus{vote1, vote2, vote3}

	// Run signature set generation function
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := SignatureSetGeneration(ctx); err != nil {
			t.Fatal(err)
		}

		wg.Done()
	}()

	// Send votes, and then block until function returns
	// The order should not matter, as the highest stake will win
	for _, vote := range allVotes {
		// Do this to avoid pointer issues during verification
		buf := new(bytes.Buffer)
		if err := vote.Encode(buf); err != nil {
			t.Fatal(err)
		}

		msg := &payload.MsgConsensus{}
		if err := msg.Decode(buf); err != nil {
			t.Fatal(err)
		}

		ctx.SigSetCandidateChan <- msg
	}
	wg.Wait()

	// We should now have otherVotes
	assert.Equal(t, otherVotes, ctx.SigSetVotes)

	// Reset step timer
	stepTime = 20 * time.Second
}

func newSigSetCandidate(c *Context, weight uint64, votes []*consensusmsg.Vote) (*payload.MsgConsensus, error) {
	if weight < MinimumStake {
		return nil, errors.New("weight too low, will result in no votes")
	}

	// Create context
	keys, _ := NewRandKeys()
	ctx, err := NewContext(0, 0, c.W, c.Round, c.Seed, c.Magic, keys)
	if err != nil {
		return nil, err
	}

	// Populate mappings on passed context
	c.NodeWeights[hex.EncodeToString([]byte(*keys.EdPubKey))] = weight
	c.NodeBLS[hex.EncodeToString(keys.BLSPubKey.Marshal())] = []byte(*keys.EdPubKey)

	// Populate new context fields
	ctx.weight = weight
	ctx.LastHeader = c.LastHeader
	ctx.BlockHash = c.BlockHash
	ctx.Step = c.Step
	ctx.SigSetVotes = votes

	// Run sortition
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.Step,
	}

	if err := sortition(ctx, role); err != nil {
		return nil, err
	}

	if ctx.votes > 0 {
		// Create payload, signature and message
		pl, err := consensusmsg.NewSigSetCandidate(ctx.BlockHash, ctx.SigSetVotes,
			ctx.Keys.BLSPubKey.Marshal(), ctx.Score)
		if err != nil {
			return nil, err
		}

		sigEd, err := createSignature(ctx, pl)
		msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
			ctx.Step, sigEd, []byte(*ctx.Keys.EdPubKey), pl)
		if err != nil {
			return nil, err
		}

		return msg, nil
	}

	return nil, nil
}
