package generation_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/sortition"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

// This test will test both signature set generation and signature set collection.
func TestSignatureSetGeneration(t *testing.T) {
	// Put step timer down to avoid long waiting times
	consensus.StepTime = 3 * time.Second

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
	voteSet, _, err := createVotesAndMsgs(ctx, 15)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetVotes = voteSet
	ctx.Weight = 200
	ctx.VoteLimit = 20

	// Run sortition
	role := &sortition.Role{
		Part:  "committee",
		Round: ctx.Round,
		Step:  ctx.Step,
	}

	score, err := sortition.CalcScore(ctx.Seed, ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, role)
	if err != nil {
		t.Fatal(err)
	}

	ctx.Score = score
	votes, err := sortition.Prove(ctx.Score, ctx.Threshold, ctx.Weight, ctx.W)
	if err != nil {
		t.Fatal(err)
	}

	ctx.Votes = votes

	// Shuffle vote set
	var otherVotes []*consensusmsg.Vote
	otherVotes = append(otherVotes, voteSet...)
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
		if err := consensus.SignatureSetGeneration(ctx); err != nil {
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
	consensus.StepTime = 20 * time.Second
}

func newSigSetCandidate(c *consensus.Context, weight uint64,
	voteSet []*consensusmsg.Vote) (*payload.MsgConsensus, error) {
	if weight < sortition.MinimumStake {
		return nil, errors.New("weight too low, will result in no votes")
	}

	// Create context
	keys, _ := consensus.NewRandKeys()
	ctx, err := consensus.NewContext(0, 0, c.W, c.Round, c.Seed, c.Magic, keys)
	if err != nil {
		return nil, err
	}

	// Populate mappings on passed context
	c.NodeWeights[hex.EncodeToString([]byte(*keys.EdPubKey))] = weight
	c.NodeBLS[hex.EncodeToString(keys.BLSPubKey.Marshal())] = []byte(*keys.EdPubKey)

	// Populate new context fields
	ctx.Weight = weight
	ctx.LastHeader = c.LastHeader
	ctx.BlockHash = c.BlockHash
	ctx.Step = c.Step
	ctx.SigSetVotes = voteSet

	// Run sortition
	role := &sortition.Role{
		Part:  "committee",
		Round: ctx.Round,
		Step:  ctx.Step,
	}

	score, err := sortition.CalcScore(ctx.Seed, ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, role)
	if err != nil {
		return nil, err
	}

	ctx.Score = score
	votes, err := sortition.Prove(ctx.Score, ctx.Threshold, ctx.Weight, ctx.W)
	if err != nil {
		return nil, err
	}

	ctx.Votes = votes
	if ctx.Votes == 0 {
		return nil, errors.New("no votes")
	}

	// Create payload, signature and message
	pl, err := consensusmsg.NewSigSetCandidate(ctx.BlockHash, ctx.SigSetVotes,
		ctx.Keys.BLSPubKey.Marshal(), score)
	if err != nil {
		return nil, err
	}

	sigEd, err := consensus.CreateSignature(ctx, pl)
	msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
		ctx.Step, sigEd, []byte(*ctx.Keys.EdPubKey), pl)
	if err != nil {
		return nil, err
	}

	return msg, nil
}
