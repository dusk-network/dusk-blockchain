package reduction_test

import (
	"encoding/hex"
	"math/rand"
	"sync"
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"

	"github.com/stretchr/testify/assert"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

// Test if votes during block reduction happen successfully.
func TestBlockReductionVote(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	ctx.Weight = 500

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

	ctx.CandidateBlock = &block.Block{}

	// Set ourselves as committee member
	pkEd := hex.EncodeToString(ctx.Keys.EdPubKeyBytes())
	ctx.NodeWeights[pkEd] = 500
	if err := ctx.Committee.AddMember(ctx.Keys.EdPubKeyBytes()); err != nil {
		t.Fatal(err)
	}

	// Start block reduction with us as the only committee member
	if err := reduction.Block(ctx); err != nil {
		t.Fatal(err)
	}
}

// Test the Block function with many votes coming in.
func TestBlockReductionDecisive(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	ctx.Weight = 500

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

	ctx.CandidateBlock = &block.Block{}

	// Set ourselves as committee member
	pkEd := hex.EncodeToString(ctx.Keys.EdPubKeyBytes())
	ctx.NodeWeights[pkEd] = 500
	if err := ctx.Committee.AddMember(ctx.Keys.EdPubKeyBytes()); err != nil {
		t.Fatal(err)
	}

	// Make 50 votes and send them to the channel beforehand
	otherVotes := make([]*payload.MsgConsensus, 0)
	for i := 0; i < 50; i++ {
		msg, msg2, err := newVoteReduction(ctx, candidateBlock)
		if err != nil {
			t.Fatal(err)
		}

		ctx.BlockReductionChan <- msg
		otherVotes = append(otherVotes, msg2)
	}

	for _, v := range otherVotes {
		ctx.BlockReductionChan <- v
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := reduction.Block(ctx); err != nil {
			t.Fatal(err)
		}

		wg.Done()
	}()

	// Wait until done
	wg.Wait()

	// Same block hash should have come out
	assert.Equal(t, candidateBlock, ctx.BlockHash)
}

// Test the Block function with many votes for a different block
// than the one we know.
func TestBlockReductionOtherBlock(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	ctx.Weight = 500

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

	ctx.CandidateBlock = &block.Block{}

	// Set ourselves as committee member
	pkEd := hex.EncodeToString(ctx.Keys.EdPubKeyBytes())
	ctx.NodeWeights[pkEd] = 500
	if err := ctx.Committee.AddMember(ctx.Keys.EdPubKeyBytes()); err != nil {
		t.Fatal(err)
	}

	// Make another block hash that voters will vote on
	otherBlock, _ := crypto.RandEntropy(32)

	// Make 50 votes and send them to the channel beforehand
	otherVotes := make([]*payload.MsgConsensus, 0)
	for i := 0; i < 50; i++ {
		msg, msg2, err := newVoteReduction(ctx, otherBlock)
		if err != nil {
			t.Fatal(err)
		}

		ctx.BlockReductionChan <- msg
		otherVotes = append(otherVotes, msg2)
	}

	for _, v := range otherVotes {
		ctx.BlockReductionChan <- v
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := reduction.Block(ctx); err != nil {
			t.Fatal(err)
		}

		wg.Done()
	}()

	// Wait until done
	wg.Wait()

	// Other block hash should have come out
	assert.Equal(t, otherBlock, ctx.BlockHash)
}

// Test the Block function with the fallback value.
func TestBlockReductionFallback(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	ctx.Weight = 500

	// Set ourselves as committee member
	pkEd := hex.EncodeToString(ctx.Keys.EdPubKeyBytes())
	ctx.NodeWeights[pkEd] = 500
	if err := ctx.Committee.AddMember(ctx.Keys.EdPubKeyBytes()); err != nil {
		t.Fatal(err)
	}

	// Make 50 votes and send them to the channel beforehand
	otherVotes := make([]*payload.MsgConsensus, 0)
	for i := 0; i < 50; i++ {
		msg, msg2, err := newVoteReduction(ctx, make([]byte, 32))
		if err != nil {
			t.Fatal(err)
		}

		ctx.BlockReductionChan <- msg
		otherVotes = append(otherVotes, msg2)
	}

	for _, v := range otherVotes {
		ctx.BlockReductionChan <- v
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := reduction.Block(ctx); err != nil {
			t.Fatal(err)
		}

		wg.Done()
	}()

	// Wait until done
	wg.Wait()

	// Block hash should be nil
	assert.Equal(t, make([]byte, 32), ctx.BlockHash)
}

// Test Block function with a low amount of votes coming in.
func TestBlockReductionIndecisive(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	ctx.Weight = 500

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

	ctx.CandidateBlock = &block.Block{}

	// Set ourselves as committee member
	pkEd := hex.EncodeToString(ctx.Keys.EdPubKeyBytes())
	ctx.NodeWeights[pkEd] = 500
	if err := ctx.Committee.AddMember(ctx.Keys.EdPubKeyBytes()); err != nil {
		t.Fatal(err)
	}

	// Make 50 votes without sending them (will increase our committee size without
	// voting)
	for i := 0; i < 50; i++ {
		weight := rand.Intn(10000)
		weight += 100 // Avoid stakes being too low to participate
		if _, _, err := newVoteReduction(ctx, candidateBlock); err != nil {
			t.Fatal(err)
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := reduction.Block(ctx); err != nil {
			t.Fatal(err)
		}

		wg.Done()
	}()

	// Wait until done
	wg.Wait()

	// Empty block hash should have come out
	assert.Equal(t, make([]byte, 32), ctx.BlockHash)
}

// Convenience function to generate a vote for the reduction phase,
// to emulate a received MsgReduction over the wire
func newVoteReduction(c *user.Context, blockHash []byte) (*payload.MsgConsensus,
	*payload.MsgConsensus, error) {
	// Create context
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, c.W, c.Round, c.Seed, c.Magic, keys)
	if err != nil {
		return nil, nil, err
	}

	weight := rand.Intn(10000)
	c.W += uint64(weight)

	// Populate mappings on passed context
	c.NodeWeights[hex.EncodeToString(keys.EdPubKeyBytes())] = uint64(weight)
	c.NodeBLS[hex.EncodeToString(keys.BLSPubKey.Marshal())] = keys.EdPubKeyBytes()

	// Populate new context fields
	ctx.Weight = uint64(weight)
	ctx.LastHeader = c.LastHeader
	ctx.BlockHash = blockHash
	ctx.BlockStep = c.BlockStep

	// Add to our committee
	if err := ctx.Committee.AddMember(ctx.Keys.EdPubKeyBytes()); err != nil {
		return nil, nil, err
	}

	// Sign block hash with BLS
	sigBLS, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, blockHash)
	if err != nil {
		return nil, nil, err
	}

	// Create reduction payload to gossip
	pl, err := consensusmsg.NewBlockReduction(blockHash, sigBLS, ctx.Keys.BLSPubKey.Marshal())
	if err != nil {
		return nil, nil, err
	}

	sigEd, err := ctx.CreateSignature(pl, ctx.BlockStep)
	if err != nil {
		return nil, nil, err
	}

	msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
		ctx.BlockStep, sigEd, ctx.Keys.EdPubKeyBytes(), pl)
	if err != nil {
		return nil, nil, err
	}

	sigEd2, err := ctx.CreateSignature(pl, ctx.BlockStep+1)
	if err != nil {
		return nil, nil, err
	}

	msg2, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
		ctx.BlockStep+1, sigEd2, ctx.Keys.EdPubKeyBytes(), pl)
	if err != nil {
		return nil, nil, err
	}

	return msg, msg2, nil
}
