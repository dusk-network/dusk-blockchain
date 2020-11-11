package chain

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	assert "github.com/stretchr/testify/require"
)

// The sequencer needs to only provide a slice of blocks
// that succeed each other. Gaps should terminate the `provideSuccessors`
// function.
func TestSequencer(t *testing.T) {
	seq := newSequencer()
	blks := make([]block.Block, 0)
	for i := 1; i < 7; i++ {
		if i == 4 {
			continue
		}

		blk := helper.RandomBlock(uint64(i), 1)
		blks = append(blks, *blk)
	}

	for _, blk := range blks {
		seq.add(blk)
	}

	// Asking for successors for blk 0 should give us blocks 1-3
	blk := helper.RandomBlock(0, 1)
	successors := seq.provideSuccessors(*blk)
	assert.True(t, len(successors) == 4)
	assert.True(t, blk.Equals(&successors[0]))
	for i := 1; i < 4; i++ {
		b := &blks[i-1]
		assert.True(t, b.Equals(&successors[i]))
	}

	// sequencer should only have block 5 and 6
	for i := 0; i < 7; i++ {
		if i == 5 || i == 6 {
			assert.NotEmpty(t, seq.blockPool[uint64(i)])
			continue
		}

		assert.Empty(t, seq.blockPool[uint64(i)])
	}
}
