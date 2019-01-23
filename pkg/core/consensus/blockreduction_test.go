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
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

// TODO: Test vote counter/signature verifier with faulty votes once
// signature libraries are implemented into context.

func TestProcessMsgReduction(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	// Create a reduction phase voting message and get their amount of votes
	emptyBlock, err := payload.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	votes, msg, err := newVoteReduction(ctx, 500, emptyBlock.Header.Hash)
	if err != nil {
		t.Fatal(err)
	}

	// Process the message
	valid, retVotes, err := processMsg(ctx, msg)
	if err != nil {
		t.Fatal(err)
	}

	// Message should be valid
	if !valid {
		t.Fatal("message was not valid")
	}

	// Votes should be equal
	assert.Equal(t, votes, retVotes)
}

// Test functionality of vote counting with a clear outcome
func TestReductionVoteCountDecisive(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.step,
	}

	// Set stake weight and vote limit, and generate a score
	ctx.weight = 500
	ctx.VoteLimit = 20
	if err := sortition(ctx, role); err != nil {
		t.Fatal(err)
	}

	// Set up voting phase
	emptyBlock, err := payload.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

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
	ctx.msgs <- msg
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

	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.step,
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
}

// BlockReduction test scenarios

// Test the BlockReduction function with many votes coming in.
func TestBlockReductionDecisive(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	ctx.VoteLimit = 10000
	ctx.weight = 500

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

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

				ctx.msgs <- msg
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

	ctx.VoteLimit = 10000
	ctx.weight = 500

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

	otherBlock, _ := crypto.RandEntropy(32)

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

				ctx.msgs <- msg
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

	ctx.VoteLimit = 10000
	ctx.weight = 500

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

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

				ctx.msgs <- msg
			}
		}
	}()

	if err := BlockReduction(ctx); err != nil {
		t.Fatal(err)
	}

	q <- true

	// Empty block hash should have come out
	assert.NotEqual(t, candidateBlock, ctx.BlockHash)
}

// Convenience function to generate a vote for the reduction phase,
// to emulate a received MsgReduction over the wire
func newVoteReduction(c *Context, weight uint64, blockHash []byte) (uint64, *payload.MsgConsensus, error) {
	if weight < 100 {
		return 0, nil, errors.New("weight too low, will result in no votes")
	}

	// Create context
	keys, _ := NewRandKeys()
	ctx, err := NewProvisionerContext(c.W, c.Round, c.Seed, c.Magic, keys)
	if err != nil {
		return 0, nil, err
	}

	c.NodeWeights[hex.EncodeToString([]byte(*keys.EdPubKey))] = weight
	ctx.weight = weight
	ctx.LastHeader = c.LastHeader
	ctx.BlockHash = blockHash

	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.step,
	}

	if err := sortition(ctx, role); err != nil {
		return 0, nil, err
	}

	if ctx.votes > 0 {
		// Sign block hash with BLS
		sigBLS, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, blockHash)
		if err != nil {
			return 0, nil, err
		}

		// Create reduction payload to gossip
		blsPubBytes := ctx.Keys.BLSPubKey.Marshal()[:32] // TODO: figure out why the length is wrong
		pl, err := consensusmsg.NewReduction(ctx.Score, ctx.step, blockHash, sigBLS, blsPubBytes)
		if err != nil {
			return 0, nil, err
		}

		sigEd, err := createSignature(ctx, pl)
		msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash, sigEd,
			[]byte(*ctx.Keys.EdPubKey), pl)
		if err != nil {
			return 0, nil, err
		}

		if err := ctx.SendMessage(ctx.Magic, msg); err != nil {
			return 0, nil, err
		}

		return ctx.votes, msg, nil
	}

	return 0, nil, nil
}
