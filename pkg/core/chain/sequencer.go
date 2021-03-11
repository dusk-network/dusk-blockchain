// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package chain

import (
	"encoding/hex"
	"errors"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/sirupsen/logrus"
)

// The sequencer is used to order incoming blocks and provide them
// in the correct order to the Chain when synchronizing.
// NOTE: the sequencer is not synchronized, as it is used by the Chain
// directly during the acceptance procedure. The mutex in this procedure
// should be sufficient to guard this map.
type sequencer struct {
	lock      sync.RWMutex
	blockPool map[uint64]block.Block
}

func newSequencer() *sequencer {
	return &sequencer{blockPool: make(map[uint64]block.Block)}
}

func (s *sequencer) add(blk block.Block) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.blockPool[blk.Header.Height] = blk
}

func (s *sequencer) get(height uint64) (block.Block, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	blk, ok := s.blockPool[height]
	if !ok {
		return block.Block{}, errors.New("block not found")
	}

	return blk, nil
}

func (s *sequencer) remove(height uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.blockPool, height)
}

// cleanup removes all blocks that are lower than currentHeight.
func (s *sequencer) cleanup(currentHeight uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for height := range s.blockPool {
		if height < currentHeight {
			delete(s.blockPool, height)
		}
	}
}

// Provide successive blocks to the given height. Once a gap is detected, the loop
// quits and returns a set of blocks.
func (s *sequencer) provideSuccessors(blk block.Block) []block.Block {
	blks := []block.Block{blk}

	for i := blk.Header.Height + 1; ; i++ {
		blk, err := s.get(i)
		if err != nil {
			return blks
		}

		blks = append(blks, blk)

		s.remove(i)
	}
}

// dump report a log entry with current state of sequencer.
func (s *sequencer) dump() {
	if logrus.GetLevel() != logrus.TraceLevel {
		return
	}

	s.lock.RLock()
	defer s.lock.RUnlock()

	for height := range s.blockPool {
		blk, err := s.get(height)
		if err == nil {
			log.WithField("hash", hex.EncodeToString(blk.Header.Hash)).
				WithField("height", blk.Header.Height).
				Trace("sequencer item")
		}
	}
}
