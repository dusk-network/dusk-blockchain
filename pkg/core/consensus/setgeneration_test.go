package consensus

import (
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

	hash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = hash
	votes, err := createVoteSet(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetVotes = votes
	ctx.weight = 500
	ctx.VoteLimit = 20
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.Step,
	}

	// Run sortition
	if err := sortition(ctx, role); err != nil {
		t.Fatal(err)
	}

	// Create votes
	_, vote1, err := newSigSetCandidate(ctx, 500)
	if err != nil {
		t.Fatal(err)
	}

	_, vote2, err := newSigSetCandidate(ctx, 800)
	if err != nil {
		t.Fatal(err)
	}

	// We should end up with the vote set of the node with the highest stake.
	// So, we save it here.
	otherVotes, vote3, err := newSigSetCandidate(ctx, 1500)
	if err != nil {
		t.Fatal(err)
	}

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
	ctx.SigSetCandidateChan <- vote1
	ctx.SigSetCandidateChan <- vote3
	ctx.SigSetCandidateChan <- vote2
	wg.Wait()

	// We should now have vote3's vote set
	assert.Equal(t, otherVotes, ctx.SigSetVotes)

	// Reset step timer
	stepTime = 20 * time.Second
}

func newSigSetCandidate(c *Context, weight uint64) ([]*consensusmsg.Vote,
	*payload.MsgConsensus, error) {
	if weight < MinimumStake {
		return nil, nil, errors.New("weight too low, will result in no votes")
	}

	// Create context
	keys, _ := NewRandKeys()
	ctx, err := NewContext(0, 0, c.W, c.Round, c.Seed, c.Magic, keys)
	if err != nil {
		return nil, nil, err
	}

	c.NodeWeights[hex.EncodeToString([]byte(*keys.EdPubKey))] = weight
	c.NodeBLS[hex.EncodeToString(keys.BLSPubKey.Marshal())] = []byte(*keys.EdPubKey)
	ctx.weight = weight
	ctx.LastHeader = c.LastHeader
	ctx.BlockHash = c.BlockHash
	ctx.Step = c.Step
	votes, err := createVoteSet(c, 50)
	if err != nil {
		return nil, nil, err
	}

	ctx.SigSetVotes = votes

	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.Step,
	}

	if err := sortition(ctx, role); err != nil {
		return nil, nil, err
	}

	if ctx.votes > 0 {
		pl, err := consensusmsg.NewSigSetCandidate(ctx.BlockHash, ctx.SigSetVotes,
			ctx.Keys.BLSPubKey.Marshal(), ctx.Score)
		if err != nil {
			return nil, nil, err
		}

		sigEd, err := createSignature(ctx, pl)
		msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
			ctx.Step, sigEd, []byte(*ctx.Keys.EdPubKey), pl)
		if err != nil {
			return nil, nil, err
		}

		return votes, msg, nil
	}

	return nil, nil, nil
}
