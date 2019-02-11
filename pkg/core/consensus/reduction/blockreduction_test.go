package reduction_test

import (
	"encoding/hex"
	"math/rand"
	"sync"
	"testing"
	"time"

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
	// Lower step timer to reduce waiting
	user.StepTime = 1 * time.Second

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

	hashStr := hex.EncodeToString(candidateBlock)
	ctx.CandidateBlocks[hashStr] = &block.Block{}

	// Set ourselves as committee member
	pkEd := hex.EncodeToString([]byte(*ctx.Keys.EdPubKey))
	ctx.NodeWeights[pkEd] = 500
	ctx.Committee = append(ctx.Committee, []byte(*ctx.Keys.EdPubKey))

	// Start block reduction with us as the only committee member
	if err := reduction.Block(ctx); err != nil {
		t.Fatal(err)
	}

	// Reset step timer
	user.StepTime = 20 * time.Second
}

// Test the Block function with many votes coming in.
func TestBlockReductionDecisive(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	ctx.Weight = 500

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

	hashStr := hex.EncodeToString(candidateBlock)
	ctx.CandidateBlocks[hashStr] = &block.Block{}

	// Set ourselves as committee member
	pkEd := hex.EncodeToString([]byte(*ctx.Keys.EdPubKey))
	ctx.NodeWeights[pkEd] = 500
	ctx.Committee = append(ctx.Committee, []byte(*ctx.Keys.EdPubKey))

	// Make 50 votes and send them to the channel beforehand
	for i := 0; i < 50; i++ {
		weight := rand.Intn(10000)
		weight += 100 // Avoid stakes being too low to participate
		msg, err := newVoteReduction(ctx, candidateBlock)
		if err != nil {
			t.Fatal(err)
		}

		ctx.BlockReductionChan <- msg
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
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	ctx.Weight = 500

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

	candidateHashStr := hex.EncodeToString(candidateBlock)
	ctx.CandidateBlocks[candidateHashStr] = &block.Block{}

	// Set ourselves as committee member
	pkEd := hex.EncodeToString([]byte(*ctx.Keys.EdPubKey))
	ctx.NodeWeights[pkEd] = 500
	ctx.Committee = append(ctx.Committee, []byte(*ctx.Keys.EdPubKey))

	// Make another block hash that voters will vote on
	otherBlock, _ := crypto.RandEntropy(32)
	otherHashStr := hex.EncodeToString(otherBlock)
	ctx.CandidateBlocks[otherHashStr] = &block.Block{}

	// Make 50 votes and send them to the channel beforehand
	for i := 0; i < 50; i++ {
		msg, err := newVoteReduction(ctx, otherBlock)
		if err != nil {
			t.Fatal(err)
		}

		ctx.BlockReductionChan <- msg
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
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	ctx.Weight = 500

	// Set ourselves as committee member
	pkEd := hex.EncodeToString([]byte(*ctx.Keys.EdPubKey))
	ctx.NodeWeights[pkEd] = 500
	ctx.Committee = append(ctx.Committee, []byte(*ctx.Keys.EdPubKey))

	// Make 50 fallback votes and send them to the channel beforehand
	for i := 0; i < 50; i++ {
		msg, err := newVoteReduction(ctx, make([]byte, 32))
		if err != nil {
			t.Fatal(err)
		}

		ctx.BlockReductionChan <- msg
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
	assert.Nil(t, ctx.BlockHash)
}

// Test Block function with a low amount of votes coming in.
func TestBlockReductionIndecisive(t *testing.T) {
	// Create context
	seed, _ := crypto.RandEntropy(32)
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, 500000, 15000, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Set basic fields on context
	ctx.Weight = 500

	candidateBlock, _ := crypto.RandEntropy(32)
	ctx.BlockHash = candidateBlock

	hashStr := hex.EncodeToString(candidateBlock)
	ctx.CandidateBlocks[hashStr] = &block.Block{}

	// Set ourselves as committee member
	pkEd := hex.EncodeToString([]byte(*ctx.Keys.EdPubKey))
	ctx.NodeWeights[pkEd] = 500
	ctx.Committee = append(ctx.Committee, []byte(*ctx.Keys.EdPubKey))

	// Adjust timer to reduce waiting times
	user.StepTime = 1 * time.Second

	// Make 50 votes without sending them (will increase our committee size without
	// voting)
	for i := 0; i < 50; i++ {
		weight := rand.Intn(10000)
		weight += 100 // Avoid stakes being too low to participate
		if _, err := newVoteReduction(ctx, candidateBlock); err != nil {
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
	assert.Nil(t, ctx.BlockHash)

	// Reset step timer
	user.StepTime = 20 * time.Second
}

// Convenience function to generate a vote for the reduction phase,
// to emulate a received MsgReduction over the wire
func newVoteReduction(c *user.Context, blockHash []byte) (*payload.MsgConsensus, error) {
	// Create context
	keys, _ := user.NewRandKeys()
	ctx, err := user.NewContext(0, 0, c.W, c.Round, c.Seed, c.Magic, keys)
	if err != nil {
		return nil, err
	}

	weight := rand.Intn(10000)

	// Populate mappings on passed context
	c.NodeWeights[hex.EncodeToString([]byte(*keys.EdPubKey))] = uint64(weight)
	c.NodeBLS[hex.EncodeToString(keys.BLSPubKey.Marshal())] = []byte(*keys.EdPubKey)

	// Populate new context fields
	ctx.Weight = uint64(weight)
	ctx.LastHeader = c.LastHeader
	ctx.BlockHash = blockHash
	ctx.Step = c.Step

	// Add to our committee, so they get at least one vote
	c.CurrentCommittee = append(c.CurrentCommittee, []byte(*keys.EdPubKey))

	// Sign block hash with BLS
	sigBLS, err := ctx.BLSSign(ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, blockHash)
	if err != nil {
		return nil, err
	}

	// Create reduction payload to gossip
	pl, err := consensusmsg.NewBlockReduction(blockHash, sigBLS, ctx.Keys.BLSPubKey.Marshal())
	if err != nil {
		return nil, err
	}

	sigEd, err := ctx.CreateSignature(pl)
	if err != nil {
		return nil, err
	}

	msg, err := payload.NewMsgConsensus(ctx.Version, ctx.Round, ctx.LastHeader.Hash,
		ctx.Step, sigEd, []byte(*ctx.Keys.EdPubKey), pl)
	if err != nil {
		return nil, err
	}

	return msg, nil
}
