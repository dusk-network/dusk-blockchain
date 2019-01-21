package consensus

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
	// Create sortition
	seed, _ := crypto.RandEntropy(32)
	totalWeight := uint64(500000)
	round := uint64(150000)

	votes, score, pk, err := runSortition(400, totalWeight, round, seed)
	if err != nil {
		t.Fatal(err)
	}

	// Create our own context to compare it with
	keys, _ := NewRandKeys()
	ctx, err := NewProvisionerContext(totalWeight, round, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create role
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.step,
	}

	// Now verify sortition
	retVotes, err := verifySortition(ctx, score, pk, role, 400)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, votes, retVotes)
}

// Implement this once signatures are done
/*func TestVerifyWrongSortition(t *testing.T) {
	// Create sortition
	seed, _ := crypto.RandEntropy(32)
	totalWeight := uint64(500000)
	round := uint64(150000)

	votes, score, pk, err := runSortition(400, totalWeight, round, seed)
	if err != nil {
		t.Fatal(err)
	}

	// Create our own context to compare it with
	keys, _ := NewRandKeys()
	ctx, err := NewProvisionerContext(totalWeight, round, seed, protocol.TestNet, keys)
	if err != nil {
		t.Fatal(err)
	}

	// Create role
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.step,
	}

	// Now verify sortition with a spoofed stake
	retVotes, err := verifySortition(ctx, score, pk, role, 200)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, retVotes, 0)
	assert.NotEqual(t, retVotes, votes)
}*/

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
	keys, _ := NewRandKeys()
	ctx, err := NewProvisionerContext(totalWeight, round, seed, protocol.TestNet, keys)
	if err != nil {
		return 0, nil, nil, err
	}

	// Set weight
	ctx.weight = weight

	// Create role
	role := &role{
		part:  "committee",
		round: ctx.Round,
		step:  ctx.step,
	}

	if err := sortition(ctx, role); err != nil {
		return 0, nil, nil, err
	}

	blsPubBytes, err := ctx.Keys.BLSPubKey.MarshalBinary()
	if err != nil {
		return 0, nil, nil, err
	}

	return ctx.votes, ctx.Score, blsPubBytes, nil
}
