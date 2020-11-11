package chain

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
)

// The sequencer is used to order incoming blocks and provide them
// in the correct order to the Chain when synchronizing.
// NOTE: the sequencer is not synchronized, as it is used by the Chain
// directly during the acceptance procedure. The mutex in this procedure
// should be sufficient to guard this map.
type sequencer struct {
	blockPool map[uint64]block.Block
}

func newSequencer() *sequencer {
	return &sequencer{blockPool: make(map[uint64]block.Block)}
}

func (s *sequencer) add(blk block.Block) {
	s.blockPool[blk.Header.Height] = blk
}

// Provide successive blocks to the given height. Once a gap is detected, the loop
// quits and returns a set of blocks.
func (s *sequencer) provideSuccessors(blk block.Block) []block.Block {
	blks := []block.Block{blk}
	for i := blk.Header.Height + 1; ; i++ {
		blk, ok := s.blockPool[i]
		if !ok {
			return blks
		}

		blks = append(blks, blk)
		delete(s.blockPool, i)
	}
}
