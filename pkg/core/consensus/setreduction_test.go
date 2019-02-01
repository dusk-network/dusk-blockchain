package consensus

import (
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

func TestSignatureSetVoteCountDecisive(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Create role for sortition
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.Step,
	}

	// Set stake weight and vote limit, and generate a score
	ctx.weight = 500
	ctx.VoteLimit = 20
	if err := sortition(ctx, role); err != nil {
		t.Fatal(err)
	}

	// Create dummy values
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	setHash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetHash = setHash
	ctx.BlockHash = emptyBlock.Header.Hash
	_, msg, err := newVoteSigSet(ctx, 400, emptyBlock.Header.Hash, setHash)
	if err != nil {
		t.Fatal(err)
	}

	// Start counting votes
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := countVotesSigSet(ctx); err != nil {
			t.Fatal(err)
		}

		wg.Done()
	}()

	// Send the vote and block until the vote counting function returns
	ctx.SigSetVoteChan <- msg
	wg.Wait()

	// Set hash should be the same after receiving vote
	assert.Equal(t, setHash, ctx.SigSetHash)
}

func TestSignatureSetVoteCountIndecisive(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Create role for sortition
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.Step,
	}

	// Set stake weight and vote limit, and generate a score
	ctx.weight = 500
	ctx.VoteLimit = 20
	if err := sortition(ctx, role); err != nil {
		t.Fatal(err)
	}

	// Adjust timer to reduce waiting times
	stepTime = 1 * time.Second

	// Let the timer run out
	if err := countVotesSigSet(ctx); err != nil {
		t.Fatal(err)
	}

	// Set hash should be the same after receiving vote
	assert.Nil(t, ctx.SigSetHash)

	// Reset step timer
	stepTime = 20 * time.Second
}

func TestSignatureSetReductionDecisive(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	ctx.weight = 500
	ctx.VoteLimit = 100
	block, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = block

	// Run sortition
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.Step,
	}

	if err := sortition(ctx, role); err != nil {
		t.Fatal(err)
	}

	// Create vote set
	votes, err := createVoteSet(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetVotes = votes

	// Run signature set reduction function
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := SignatureSetReduction(ctx); err != nil {
			t.Fatal(err)
		}

		wg.Done()
	}()

	// Wait one second..
	time.Sleep(1 * time.Second)

	// SigSetHash should now be set
	// Make vote that will win by a large margin
	_, vote1, err := newVoteSigSet(ctx, 1000, block, ctx.SigSetHash)
	if err != nil {
		t.Fatal(err)
	}

	// Log current SigSetHash
	var currHash []byte
	currHash = append(currHash, ctx.SigSetHash...)

	// Send vote
	ctx.SigSetVoteChan <- vote1

	// Wait one second..
	time.Sleep(1 * time.Second)

	// Make new vote, send it, and wait for function to return
	_, vote2, err := newVoteSigSet(ctx, 1000, block, ctx.SigSetHash)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetVoteChan <- vote2
	wg.Wait()

	// Function should have returned and we should have the same sigsethash
	assert.Equal(t, currHash, ctx.SigSetHash)
}

func TestSignatureSetReductionIndecisive(t *testing.T) {
	// Adjust timer to reduce waiting times
	stepTime = 100 * time.Millisecond

	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	ctx.weight = 500
	ctx.VoteLimit = 100
	block, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	ctx.BlockHash = block

	// Run sortition
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.Step,
	}

	if err := sortition(ctx, role); err != nil {
		t.Fatal(err)
	}

	// Create vote set
	votes, err := createVoteSet(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}

	ctx.SigSetVotes = votes

	// The function will loop as long as it doesn't receive a convincing majority
	// vote. After hitting MaxSteps, the function will return.
	if err := SignatureSetReduction(ctx); err != nil {
		t.Fatal(err)
	}

	// Function should have returned and we should have a nil sigsethash
	assert.Nil(t, ctx.SigSetHash)

	// Reset step timer
	stepTime = 20 * time.Second
}

func newVoteSigSet(c *Context, weight uint64, winningBlock, setHash []byte) (uint64, *payload.MsgConsensus, error) {
	if weight < MinimumStake {
		return 0, nil, errors.New("weight too low, will result in no votes")
	}

	// Create context
	keys, _ := NewRandKeys()
	ctx, err := NewContext(0, 0, c.W, c.Round, c.Seed, c.Magic, keys)
	if err != nil {
		return 0, nil, err
	}

	// Populate mappings on passed context
	c.NodeWeights[hex.EncodeToString([]byte(*keys.EdPubKey))] = weight
	c.NodeBLS[hex.EncodeToString(keys.BLSPubKey.Marshal())] = []byte(*keys.EdPubKey)

	// Populate new context fields
	ctx.weight = weight
	ctx.LastHeader = c.LastHeader
	ctx.SigSetHash = setHash
	ctx.BlockHash = winningBlock
	ctx.Step = c.Step

	// Run sortition
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.Step,
	}

	if err := sortition(ctx, role); err != nil {
		return 0, nil, err
	}

	if ctx.votes > 0 {
		// Sign sig set with BLS
		sigBLS, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, ctx.SigSetHash)
		if err != nil {
			return 0, nil, err
		}

		// Set BLS key on context
		blsPubBytes := ctx.Keys.BLSPubKey.Marshal()
		c.NodeBLS[hex.EncodeToString([]byte(*keys.EdPubKey))] = blsPubBytes

		// Create sigsetvote payload to gossip
		pl, err := consensusmsg.NewSigSetVote(winningBlock, ctx.SigSetHash, sigBLS, blsPubBytes, ctx.Score)
		if err != nil {
			return 0, nil, err
		}

		sigEd, err := createSignature(ctx, pl)
		msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
			ctx.Step, sigEd, []byte(*ctx.Keys.EdPubKey), pl)
		if err != nil {
			return 0, nil, err
		}

		return ctx.votes, msg, nil
	}

	return 0, nil, nil
}
