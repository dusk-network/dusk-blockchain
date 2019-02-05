package sortition_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/sortition"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

// Test the sortition function on multiple stake sizes and make sure they differ.
func TestSortition(t *testing.T) {
	// Run sortitions
	votes1, err := runMultipleSortitions(200, 500000, 150000, 5)
	if err != nil {
		t.Fatal(err)
	}

	votes2, err := runMultipleSortitions(500, 500000, 150000, 5)
	if err != nil {
		t.Fatal(err)
	}

	votes3, err := runMultipleSortitions(1500, 500000, 150000, 5)
	if err != nil {
		t.Fatal(err)
	}

	// Compare between arrays
	assert.True(t, votes1[1] < votes2[1])
	assert.True(t, votes2[0] < votes3[2])

	// Make sure results differ from each other (influenced by score)
	assert.NotEqual(t, votes1[0], votes1[3])
	assert.NotEqual(t, votes2[2], votes2[4])
	assert.NotEqual(t, votes3[3], votes1[1])
}

func TestVerifySortition(t *testing.T) {
	// Create score
	seed, _ := crypto.RandEntropy(32)
	totalWeight := uint64(500000)
	round := uint64(150000)

	votes, score, pk, err := runSortition(400, totalWeight, round, seed)
	if err != nil {
		t.Fatal(err)
	}

	// Create our own context to compare it with
	keys, _ := consensus.NewRandKeys()
	ctx, err := consensus.NewContext(0, 0, totalWeight, round, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create role
	role := &sortition.Role{
		Part:  "committee",
		Round: ctx.Round,
		Step:  ctx.Step,
	}

	// Now verify sortition
	retVotes, err2 := sortition.Verify(ctx.Seed, score, pk, role, ctx.Threshold, 400, ctx.W)
	if err2 != nil {
		t.Fatal(err2)
	}

	if retVotes == 0 {
		t.Fatal("sortition was not valid")
	}

	assert.Equal(t, votes, retVotes)
}

// Convenience function to run sortition a number of times
func runMultipleSortitions(weight, totalWeight, round uint64, times int) ([]uint64, error) {
	var voteArray []uint64
	for i := 0; i < times; i++ {
		// Use random seed each time
		seed, _ := crypto.RandEntropy(32)

		votes, _, _, err := runSortition(weight, totalWeight, round, seed)
		if err != nil {
			return nil, err
		}

		voteArray = append(voteArray, votes)
	}

	return voteArray, nil
}

// Run sortition function with specified parameters and return context info
func runSortition(weight, totalWeight, round uint64, seed []byte) (uint64, []byte, []byte, error) {
	// Create dummy context
	keys, _ := consensus.NewRandKeys()
	ctx, err := consensus.NewContext(0, 0, totalWeight, round, seed, protocol.TestNet, keys)
	if err != nil {
		return 0, nil, nil, err
	}

	// Set weight
	ctx.Weight = weight

	// Create role
	role := &sortition.Role{
		Part:  "committee",
		Round: ctx.Round,
		Step:  ctx.Step,
	}

	score, err := sortition.CalcScore(ctx.Seed, ctx.Keys.BLSSecretKey, ctx.Keys.BLSPubKey, role)
	if err != nil {
		return 0, nil, nil, err
	}

	ctx.Score = score
	votes, err := sortition.Prove(ctx.Score, ctx.Threshold, ctx.Weight, ctx.W)
	if err != nil {
		return 0, nil, nil, err
	}

	ctx.Votes = votes
	blsPubBytes := ctx.Keys.BLSPubKey.Marshal()
	return ctx.Votes, ctx.Score, blsPubBytes, nil
}
