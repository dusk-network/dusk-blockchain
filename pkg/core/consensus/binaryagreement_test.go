package consensus

import (
	"encoding/hex"
	"errors"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
)

func TestCommonCoin(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	// Create random amount of MsgConsensus
	var msgs []*payload.MsgConsensus
	n := rand.Intn(20)
	for i := 0; i < n; i++ {
		weight := rand.Intn(2000)
		weight += 100 // Avoid getting stakes that are too low
		_, msg, err := newVoteAgreement(ctx, uint64(weight), emptyBlock.Header.Hash)
		if err != nil {
			t.Fatal(err)
		}

		msgs = append(msgs, msg)
	}

	// Run CommonCoin
	result, err := commonCoin(ctx, msgs)
	if err != nil {
		t.Fatal(err)
	}

	// result should be either 0 or 1
	if result != 0 && result != 1 {
		t.Fatal("commonCoin produced a result that was not 0 or 1")
	}
}

// Test functionality of vote counting with a clear outcome
func TestBinaryVoteCountDecisive(t *testing.T) {
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
	emptyBlock, err := block.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	_, msg, err := newVoteAgreement(ctx, 400, emptyBlock.Header.Hash)
	if err != nil {
		t.Fatal(err)
	}

	// Start listening for votes
	var msgs []*payload.MsgConsensus
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		var err error
		msgs, err = countVotesBinary(ctx)
		if err != nil {
			t.Fatal(err)
		}

		wg.Done()
	}()

	// Send the vote out, and block until the counting function returns
	ctx.msgs <- msg
	wg.Wait()

	// BlockHash should not be nil after receiving vote
	assert.NotNil(t, ctx.BlockHash)

	// msgs should contain msg
	assert.Equal(t, msgs[0], msg)
}

// Test functionality of vote counting when no clear outcome is reached
func TestBinaryVoteCountIndecisive(t *testing.T) {
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
	msgs, err := countVotesBinary(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// BlockHash should be nil after hitting time limit
	assert.Nil(t, ctx.BlockHash)

	// msgs should also be nil
	assert.Nil(t, msgs)

	// Reset step timer
	stepTime = 20 * time.Second
}

// BinaryAgreement test scenarios

// Test the BinaryAgreement function with many votes coming in.
func TestBinaryAgreementDecisive(t *testing.T) {
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	ctx.VoteLimit = 10000
	ctx.weight = 500

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

	// This should conclude the binary agreement phase fairly quick
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
				_, msg, err := newVoteAgreement(ctx, uint64(weight), candidateBlock)
				if err != nil {
					t.Fatal(err)
				}

				ctx.msgs <- msg
			}
		}
	}()

	if err := BinaryAgreement(ctx); err != nil {
		t.Fatal(err)
	}

	q <- true

	// Block hash should be the same
	assert.Equal(t, candidateBlock, ctx.BlockHash)
}

// Test the BinaryAgreement function with many votes coming in for a different
// block than the one we know.
func TestBinaryAgreementOtherBlock(t *testing.T) {
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	ctx.VoteLimit = 10000
	ctx.weight = 500

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

	otherBlock, _ := crypto.RandEntropy(32)

	// This should conclude the binary agreement phase fairly quick
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
				_, msg, err := newVoteAgreement(ctx, uint64(weight), otherBlock)
				if err != nil {
					t.Fatal(err)
				}

				ctx.msgs <- msg
			}
		}
	}()

	if err := BinaryAgreement(ctx); err != nil {
		t.Fatal(err)
	}

	q <- true

	// Other block hash should have come out
	assert.Equal(t, otherBlock, ctx.BlockHash)
}

// Test the BinaryAgreement function with little votes coming in, and triggering
// the coin-flipped-to-0 step with a non-empty block.
func TestCoinFlippedNonEmpty(t *testing.T) {
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	ctx.VoteLimit = 10000
	ctx.weight = 500

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

	// Bring stepTime down to avoid long waiting times
	stepTime = 5 * time.Second

	// This should time out the binary agreement voting phase
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
				_, msg, err := newVoteAgreement(ctx, uint64(weight), candidateBlock)
				if err != nil {
					t.Fatal(err)
				}

				ctx.msgs <- msg
			}
		}
	}()

	if err := BinaryAgreement(ctx); err != nil {
		t.Fatal(err)
	}

	q <- true

	// Block hash should be the same
	assert.Equal(t, candidateBlock, ctx.BlockHash)

	// Step should be maxSteps (coin-flipped-to-0 with a non-empty block sets ctx.step to maxSteps)
	assert.Equal(t, maxSteps, ctx.step)

	// Reset step timer
	stepTime = 20 * time.Second
}

// Test the BinaryAgreement function with little votes coming in, and triggering
// the coin-flipped-to-0 step and the coin-flipped-to-1 step with an empty block.
func TestCoinFlippedEmpty(t *testing.T) {
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	ctx.VoteLimit = 10000
	ctx.weight = 500

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock
	ctx.Empty = true

	// Bring stepTime down to avoid long waiting times
	stepTime = 5 * time.Second

	// This should time out the binary agreement voting phase
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
				_, msg, err := newVoteAgreement(ctx, uint64(weight), candidateBlock)
				if err != nil {
					t.Fatal(err)
				}

				ctx.msgs <- msg
			}
		}
	}()

	if err := BinaryAgreement(ctx); err != nil {
		t.Fatal(err)
	}

	q <- true

	// Block hash should not be the same
	assert.NotEqual(t, candidateBlock, ctx.BlockHash)

	// Step should be 2 (coin-flipped-to-1 with an empty block terminates on step 2)
	assert.Equal(t, 2, int(ctx.step))

	// ctx.Empty should be true
	assert.True(t, ctx.Empty)

	// Reset step timer
	stepTime = 20 * time.Second
}

// Convenience function to generate a vote for the binary agreement phase,
// to emulate a received MsgBinary over the wire. This function emulates an empty block.
func newVoteAgreement(c *Context, weight uint64, blockHash []byte) (uint64, *payload.MsgConsensus, error) {
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
	ctx.step = c.step

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
		sigBLS, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, blockHash)
		if err != nil {
			return 0, nil, err
		}

		// Create agreement payload to gossip
		blsPubBytes := ctx.Keys.BLSPubKey.Marshal()
		pl, err := consensusmsg.NewAgreement(ctx.Score, ctx.Empty, ctx.step, blockHash,
			sigBLS, blsPubBytes)
		if err != nil {
			return 0, nil, err
		}

		sigEd, err := createSignature(ctx, pl)
		if err != nil {
			return 0, nil, err
		}

		msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
			sigEd, []byte(*ctx.Keys.EdPubKey), pl)
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
