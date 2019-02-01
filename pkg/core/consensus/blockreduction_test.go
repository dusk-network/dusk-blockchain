package consensus

import (
	"encoding/hex"
	"errors"
	"math/rand"
	"sync"
	"testing"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"

	"github.com/stretchr/testify/assert"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// Test functionality of vote counting with a clear outcome
func TestReductionVoteCountDecisive(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Make role for sortition
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

	// Set up voting phase
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	hashStr := hex.EncodeToString(emptyBlock.Header.Hash)
	ctx.CandidateBlocks[hashStr] = &block.Block{}

	// Make a block reduction messages
	_, msg, err := newVoteReduction(ctx, 400, emptyBlock.Header.Hash)
	if err != nil {
		t.Fatal(err)
	}

	// Start listening for votes
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := countVotesReduction(ctx); err != nil {
			t.Fatal(err)
		}

		wg.Done()
	}()

	// Send the vote out, and block until the counting function returns
	ctx.ReductionChan <- msg
	wg.Wait()

	// BlockHash should not be nil after receiving vote
	assert.NotNil(t, ctx.BlockHash)
}

// Test functionality of vote counting when no clear outcome is reached
func TestReductionVoteCountIndecisive(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Make role for sortition
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
	if err := countVotesReduction(ctx); err != nil {
		t.Fatal(err)
	}

	// BlockHash should be nil after hitting time limit
	assert.Nil(t, ctx.BlockHash)

	// Reset step timer
	stepTime = 20 * time.Second
}

// Test if a repeated vote does not add to the counter
func TestReductionVoteCountDuplicates(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Make role for sortition
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.Step,
	}

	// Set stake weight and vote limit, and generate a score
	ctx.weight = 500
	ctx.VoteLimit = 2000
	if err := sortition(ctx, role); err != nil {
		t.Fatal(err)
	}

	// Set up voting phase
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	// Make block reduction message
	_, msg, err := newVoteReduction(ctx, 400, emptyBlock.Header.Hash)
	if err != nil {
		t.Fatal(err)
	}

	// Adjust timer to reduce waiting times
	stepTime = 3 * time.Second

	// Start listening for votes
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := countVotesReduction(ctx); err != nil {
			t.Fatal(err)
		}

		wg.Done()
	}()

	// Send the vote out ten times, and block until the counting function returns
	for i := 0; i < 10; i++ {
		ctx.ReductionChan <- msg
	}

	wg.Wait()

	// BlockHash should be nil after not receiving enough votes
	assert.Nil(t, ctx.BlockHash)

	// Reset step timer
	stepTime = 20 * time.Second
}

// BlockReduction test scenarios

// Test the BlockReduction function with many votes coming in.
func TestBlockReductionDecisive(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	ctx.VoteLimit = 10000
	ctx.weight = 500

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

	hashStr := hex.EncodeToString(candidateBlock)
	ctx.CandidateBlocks[hashStr] = &block.Block{}

	// Continually send reduction messages from a goroutine.
	// This should conclude the reduction phase fairly quick
	q := make(chan bool, 1)
	ticker := time.NewTicker(500 * time.Millisecond)
	go func() {
		for {
			select {
			case <-q:
				ticker.Stop()
				return
			case <-ticker.C:
				weight := rand.Intn(10000)
				weight += 100 // Avoid stakes being too low to participate
				_, msg, err := newVoteReduction(ctx, uint64(weight), candidateBlock)
				if err != nil {
					t.Fatal(err)
				}

				ctx.ReductionChan <- msg
			}
		}
	}()

	if err := BlockReduction(ctx); err != nil {
		t.Fatal(err)
	}

	q <- true

	// Same block hash should have come out
	assert.Equal(t, candidateBlock, ctx.BlockHash)
}

// Test the BlockReduction function with many votes for a different block
// than the one we know.
func TestBlockReductionOtherBlock(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	ctx.VoteLimit = 10000
	ctx.weight = 500

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

	candidateHashStr := hex.EncodeToString(candidateBlock)
	ctx.CandidateBlocks[candidateHashStr] = &block.Block{}

	// Make another block hash that voters will vote on
	otherBlock, _ := crypto.RandEntropy(32)
	otherHashStr := hex.EncodeToString(otherBlock)
	ctx.CandidateBlocks[otherHashStr] = &block.Block{}

	// Continually send reduction messages from a goroutine.
	// This should conclude the reduction phase fairly quick
	q := make(chan bool, 1)
	ticker := time.NewTicker(500 * time.Millisecond)
	go func() {
		for {
			select {
			case <-q:
				ticker.Stop()
				return
			case <-ticker.C:
				weight := rand.Intn(10000)
				weight += 100 // Avoid stakes being too low to participate
				_, msg, err := newVoteReduction(ctx, uint64(weight), otherBlock)
				if err != nil {
					t.Fatal(err)
				}

				ctx.ReductionChan <- msg
			}
		}
	}()

	if err := BlockReduction(ctx); err != nil {
		t.Fatal(err)
	}

	q <- true

	// Other block hash should have come out
	assert.Equal(t, otherBlock, ctx.BlockHash)
}

// Test BlockReduction function with a low amount of votes coming in.
func TestBlockReductionIndecisive(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	ctx.VoteLimit = 10000
	ctx.weight = 500

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

	hashStr := hex.EncodeToString(candidateBlock)
	ctx.CandidateBlocks[hashStr] = &block.Block{}

	// Adjust timer to reduce waiting times
	stepTime = 1 * time.Second

	// This should time out and change our context blockhash
	q := make(chan bool, 1)
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-q:
				ticker.Stop()
				return
			case <-ticker.C:
				weight := rand.Intn(1000)
				weight += 100 // Avoid stakes being too low to participate
				_, msg, err := newVoteReduction(ctx, uint64(weight), candidateBlock)
				if err != nil {
					t.Fatal(err)
				}

				ctx.ReductionChan <- msg
			}
		}
	}()

	if err := BlockReduction(ctx); err != nil {
		t.Fatal(err)
	}

	q <- true

	// Empty block hash should have come out
	assert.NotEqual(t, candidateBlock, ctx.BlockHash)

	// Reset step timer
	stepTime = 20 * time.Second
}

// Convenience function to generate a vote for the reduction phase,
// to emulate a received MsgReduction over the wire
func newVoteReduction(c *Context, weight uint64, blockHash []byte) (uint64, *payload.MsgConsensus, error) {
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
	ctx.BlockHash = blockHash
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
		// Sign block hash with BLS
		sigBLS, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, blockHash)
		if err != nil {
			return 0, nil, err
		}

		// Create reduction payload to gossip
		pl, err := consensusmsg.NewReduction(ctx.Score, blockHash, sigBLS, ctx.Keys.BLSPubKey.Marshal())
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
