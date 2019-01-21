package consensus

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

func TestProcessMsgBinary(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	emptyBlock, err := payload.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	// Create a binary agreement phase voting message and get their amount of votes
	votes, msg, err := newVoteBinary(ctx.Seed, 500, ctx.W, ctx.Round, ctx.LastHeader,
		emptyBlock.Header.Hash, true)
	if err != nil {
		t.Fatal(err)
	}

	// Process the message
	retVotes, _, err := processMsgBinary(ctx, msg)
	if err != nil {
		t.Fatal(err)
	}

	// Votes should be equal
	assert.Equal(t, votes, retVotes)
}

func TestCommonCoin(t *testing.T) {
	// Create context
	ctx, err := provisionerContext()
	if err != nil {
		t.Fatal(err)
	}

	emptyBlock, err := payload.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	// Create random amount of MsgBinary
	var msgs []*payload.MsgBinary
	n := rand.Intn(20)
	for i := 0; i < n; i++ {
		weight := rand.Intn(2000)
		weight += 100 // Avoid getting stakes that are too low
		_, msg, err := newVoteBinary(ctx.Seed, uint64(weight), ctx.W, ctx.Round, ctx.LastHeader,
			emptyBlock.Header.Hash, true)
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
	c := make(chan *payload.MsgBinary)
	emptyBlock, err := payload.NewEmptyBlock(ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	_, msg, err := newVoteBinary(ctx.Seed, 400, ctx.W, ctx.Round, ctx.LastHeader,
		emptyBlock.Header.Hash, true)
	if err != nil {
		t.Fatal(err)
	}

	// Start listening for votes
	var msgs []*payload.MsgBinary
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		var err error
		msgs, err = countVotesBinary(ctx, c)
		if err != nil {
			t.Fatal(err)
		}

		wg.Done()
	}()

	// Send the vote out, and block until the counting function returns
	c <- msg
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
	c := make(chan *payload.MsgBinary)
	msgs, err := countVotesBinary(ctx, c)
	if err != nil {
		t.Fatal(err)
	}

	// BlockHash should be nil after hitting time limit
	assert.Nil(t, ctx.BlockHash)

	// msgs should also be nil
	assert.Nil(t, msgs)
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
	c := make(chan *payload.MsgBinary)
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
				_, msg, err := newVoteBinary(ctx.Seed, uint64(weight), ctx.W, ctx.Round, ctx.LastHeader,
					ctx.BlockHash, ctx.Empty)
				if err != nil {
					t.Fatal(err)
				}

				c <- msg
			}
		}
	}()

	if err := BinaryAgreement(ctx, c); err != nil {
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
	c := make(chan *payload.MsgBinary)
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
				_, msg, err := newVoteBinary(ctx.Seed, uint64(weight), ctx.W, ctx.Round, ctx.LastHeader,
					otherBlock, ctx.Empty)
				if err != nil {
					t.Fatal(err)
				}

				c <- msg
			}
		}
	}()

	if err := BinaryAgreement(ctx, c); err != nil {
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
	c := make(chan *payload.MsgBinary)
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
				_, msg, err := newVoteBinary(ctx.Seed, uint64(weight), ctx.W, ctx.Round, ctx.LastHeader,
					ctx.BlockHash, ctx.Empty)
				if err != nil {
					t.Fatal(err)
				}

				c <- msg
			}
		}
	}()

	if err := BinaryAgreement(ctx, c); err != nil {
		t.Fatal(err)
	}

	q <- true

	// Block hash should be the same
	assert.Equal(t, candidateBlock, ctx.BlockHash)

	// Step should be maxSteps (coin-flipped-to-0 with a non-empty block sets ctx.step to maxSteps)
	assert.Equal(t, maxSteps, ctx.step)
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
	c := make(chan *payload.MsgBinary)
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
				_, msg, err := newVoteBinary(ctx.Seed, uint64(weight), ctx.W, ctx.Round, ctx.LastHeader,
					ctx.BlockHash, ctx.Empty)
				if err != nil {
					t.Fatal(err)
				}

				c <- msg
			}
		}
	}()

	if err := BinaryAgreement(ctx, c); err != nil {
		t.Fatal(err)
	}

	q <- true

	// Block hash should not be the same
	assert.NotEqual(t, candidateBlock, ctx.BlockHash)

	// Step should be 2 (coin-flipped-to-1 with an empty block terminates on step 2)
	assert.Equal(t, 2, int(ctx.step))

	// ctx.Empty should be true
	assert.True(t, ctx.Empty)
}

// Convenience function to generate a vote for the binary agreement phase,
// to emulate a received MsgBinary over the wire. This function emulates an empty block.
func newVoteBinary(seed []byte, weight, totalWeight, round uint64, prevHeader *payload.BlockHeader,
	blockHash []byte, empty bool) (int, *payload.MsgBinary, error) {
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
	ctx.Empty = empty

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

		msg, err := payload.NewMsgBinary(ctx.Score, ctx.Empty, ctx.BlockHash, ctx.LastHeader.Hash, sigEd,
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
