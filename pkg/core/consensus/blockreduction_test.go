package consensus

import (
	"encoding/binary"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

// TODO: Test vote counter/signature verifier with faulty votes once
// signature libraries are implemented into context.

func TestProcessMsgReduction(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := NewRandKeys()
	totalWeight := uint64(500000)
	round := uint64(150000)
	ctx, err := NewProvisionerContext(totalWeight, round, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	votes, msg, err := newVoteReduction(seed, 500, totalWeight, round, ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	retVotes, _, err := processMsgReduction(ctx, msg)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, votes, retVotes)
}

func TestReductionVoteCountDecisive(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := NewRandKeys()
	totalWeight := uint64(500000)
	round := uint64(150000)
	ctx, err := NewProvisionerContext(totalWeight, round, seed, protocol.TestNet, keys)
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

	c := make(chan *payload.MsgReduction)
	_, msg, err := newVoteReduction(seed, 400, totalWeight, round, ctx.LastHeader)
	if err != nil {
		t.Fatal(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := countVotesReduction(ctx, c); err != nil {
			t.Fatal(err)
		}

		wg.Done()
	}()

	c <- msg
	wg.Wait()

	// BlockHash should not be nil after receiving vote
	assert.NotNil(t, ctx.BlockHash)
}

func TestReductionVoteCountIndecisive(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := NewRandKeys()
	totalWeight := uint64(500000)
	round := uint64(150000)
	ctx, err := NewProvisionerContext(totalWeight, round, seed, protocol.TestNet, keys)
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

	// This will take a few seconds...
	c := make(chan *payload.MsgReduction)
	if err := countVotesReduction(ctx, c); err != nil {
		t.Fatal(err)
	}

	// BlockHash should be nil after hitting time limit
	assert.Nil(t, ctx.BlockHash)
}

func TestBlockReduction(t *testing.T) {
	//
}

// Convenience function to generate a vote for the reduction phase,
// to emulate a received MsgReduction over the wire
func newVoteReduction(seed []byte, weight, totalWeight, round uint64, prevHeader *payload.BlockHeader) (int, *payload.MsgReduction, error) {
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

	// Create empty block
	emptyBlock, err := payload.NewEmptyBlock(prevHeader)
	if err != nil {
		return 0, nil, err
	}

	ctx.BlockHash = emptyBlock.Header.Hash

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
		sigEd, err := ctx.EDSign(ctx.Keys.EdSecretKey, edMsg)
		if err != nil {
			return 0, nil, err
		}

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
