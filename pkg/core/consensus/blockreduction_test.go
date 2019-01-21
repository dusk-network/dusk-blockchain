package consensus

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"sync"
	"testing"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"

	"github.com/stretchr/testify/assert"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
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

	votes, msg, err := newVoteReduction(ctx.Seed, 500, ctx.W, ctx.Round, ctx.LastHeader, emptyBlock.Header.Hash)
	if err != nil {
		t.Fatal(err)
	}

	// Process the message
	retVotes, _, err := processMsgReduction(ctx, msg)
	if err != nil {
		t.Fatal(err)
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
	c := make(chan *payload.MsgReduction)
	emptyBlock, err := payload.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	_, msg, err := newVoteReduction(ctx.Seed, 400, ctx.W, ctx.Round, ctx.LastHeader, emptyBlock.Header.Hash)
	if err != nil {
		t.Fatal(err)
	}

	// Start listening for votes
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := countVotesReduction(ctx, c); err != nil {
			t.Fatal(err)
		}

		wg.Done()
	}()

	// Send the vote out, and block until the counting function returns
	c <- msg
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
	c := make(chan *payload.MsgReduction)
	if err := countVotesReduction(ctx, c); err != nil {
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
	c := make(chan *payload.MsgReduction)
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
				_, msg, err := newVoteReduction(ctx.Seed, uint64(weight), ctx.W, ctx.Round, ctx.LastHeader, candidateBlock)
				if err != nil {
					t.Fatal(err)
				}

				c <- msg
			}
		}
	}()

	if err := BlockReduction(ctx, c); err != nil {
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
	c := make(chan *payload.MsgReduction)
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
				_, msg, err := newVoteReduction(ctx.Seed, uint64(weight), ctx.W, ctx.Round, ctx.LastHeader, otherBlock)
				if err != nil {
					t.Fatal(err)
				}

				c <- msg
			}
		}
	}()

	if err := BlockReduction(ctx, c); err != nil {
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
	c := make(chan *payload.MsgReduction)
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
				_, msg, err := newVoteReduction(ctx.Seed, uint64(weight), ctx.W, ctx.Round, ctx.LastHeader, candidateBlock)
				if err != nil {
					t.Fatal(err)
				}

				c <- msg
			}
		}
	}()

	if err := BlockReduction(ctx, c); err != nil {
		t.Fatal(err)
	}

	q <- true

	// Empty block hash should have come out
	assert.NotEqual(t, candidateBlock, ctx.BlockHash)
}

// Convenience function to generate a vote for the reduction phase,
// to emulate a received MsgReduction over the wire
func newVoteReduction(seed []byte, weight, totalWeight, round uint64, prevHeader *payload.BlockHeader,
	blockHash []byte) (int, *payload.MsgReduction, error) {
	if weight < 100 {
		return 0, nil, errors.New("weight too low, will result in no votes")
	}

	// Create context
	keys, _ := NewRandKeys()
	ctx, err := NewProvisionerContext(totalWeight, round, seed, protocol.TestNet, keys)
	if err != nil {
		return 0, nil, err
	}

	ctx.weight = weight
	ctx.LastHeader = prevHeader
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
		sigBLS, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, ctx.BlockHash)
		if err != nil {
			return 0, nil, err
		}

		// Create message to sign with ed25519
		var edMsg []byte
		edMsg = append(edMsg, ctx.Score...)
		binary.LittleEndian.PutUint64(edMsg, ctx.Round)
		edMsg = append(edMsg, byte(ctx.step))
		edMsg = append(edMsg, ctx.LastHeader.Hash...)
		edMsg = append(edMsg, sigBLS...)

		// Sign with ed25519
		sigEd := ctx.EDSign(ctx.Keys.EdSecretKey, edMsg)

		// Create reduction message to gossip
		blsPubBytes, err := ctx.Keys.BLSPubKey.MarshalBinary()
		if err != nil {
			return 0, nil, err
		}

		msg, err := payload.NewMsgReduction(ctx.Score, ctx.BlockHash, ctx.LastHeader.Hash, sigEd,
			[]byte(*ctx.Keys.EdPubKey), sigBLS, blsPubBytes, ctx.weight, ctx.Round, ctx.step)
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
