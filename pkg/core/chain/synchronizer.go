// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package chain

import (
	"bytes"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

const (
	syncTimeout = time.Duration(5) * time.Second
)

type syncState func(currentHeight uint64, blk block.Block) ([]bytes.Buffer, error)

func (s *synchronizer) inSync(currentHeight uint64, blk block.Block) ([]bytes.Buffer, error) {
	if blk.Header.Height > currentHeight+1 {
		// If this block is from far in the future, we should start syncing mode.
		s.chain.StopBlockProduction()
		s.sequencer.add(blk)

		// Trigger timeout outSync timer. If the peer trigerring the Sync
		// procedure is dishonest, this timer should switch state back to InSync
		//
		// A peer is marked as dishonest if it cannot provide next valid
		// consecutive block within syncTimeout seconds
		s.timer = time.NewTimer(syncTimeout)

		go func() {
			_, ok := <-s.timer.C
			if !ok {
				return
			}

			// Dishonest peer.
			// TODO: Implement Peer addr temporary ban

			_ = s.chain.ProduceBlock()
			s.state = s.inSync
		}()

		s.state = s.outSync
		b, err := s.startSync(blk.Header.Height, currentHeight)
		return b, err
	}

	// Otherwise notify the chain (and the consensus loop).
	if err := s.chain.TryNextConsecutiveBlockInSync(blk); err != nil {
		return nil, err
	}

	return nil, nil
}

func (s *synchronizer) outSync(currentHeight uint64, blk block.Block) ([]bytes.Buffer, error) {
	var err error

	if blk.Header.Height > currentHeight+1 {
		// if there is a gap we add the future block to the sequencer
		s.sequencer.add(blk)

		blk, err = s.sequencer.get(currentHeight + 1)
		if err != nil {
			return nil, nil
		}
	}

	// Retrieve all successive blocks that need to be accepted
	blks := s.sequencer.provideSuccessors(blk)

	for _, blk := range blks {
		// append them all to the ledger
		if err = s.chain.TryNextConsecutiveBlockOutSync(blk); err != nil {
			log.WithError(err).Debug("could not AcceptBlock")
			return nil, err
		}

		if s.timer != nil {
			// Peer does provide a valid consecutive block
			_ = s.timer.Reset(syncTimeout)
		}

		if blk.Header.Height == s.syncTarget {
			// if we reach the target we get into sync mode
			// and trigger the consensus again
			if err = s.chain.ProduceBlock(); err != nil {
				return nil, err
			}

			if s.timer != nil {
				_ = s.timer.Stop()
				s.timer = nil
			}

			s.state = s.inSync
		}
	}

	return nil, nil
}

// synchronizer acts as the gateway for incoming blocks from the network.
// It decides how the Chain should process these blocks, and is responsible
// for requesting missing items in case of a desync.
// NOTE: The synchronizer is not thread-safe.
type synchronizer struct {
	db    database.DB
	state syncState
	*sequencer
	chain      Ledger
	syncTarget uint64

	timer *time.Timer
}

// newSynchronizer returns an initialized synchronizer, ready for use.
func newSynchronizer(db database.DB, chain Ledger) *synchronizer {
	s := &synchronizer{
		db:        db,
		sequencer: newSequencer(),
		chain:     chain,
	}

	s.state = s.inSync
	return s
}

// processBlock handles an incoming block from the network.
func (s *synchronizer) processBlock(currentHeight uint64, blk block.Block) (res []bytes.Buffer, err error) {
	// Clean up sequencer
	s.sequencer.cleanup(currentHeight)

	currState := s.state
	res, err = currState(currentHeight, blk)
	return
}

func (s *synchronizer) startSync(tipHeight, currentHeight uint64) ([]bytes.Buffer, error) {

	s.setSyncTarget(tipHeight, currentHeight+config.MaxInvBlocks)

	log.WithField("curr", currentHeight).
		WithField("tip", tipHeight).WithField("target", s.syncTarget).
		Trace("Starting syncing")

	var hash []byte

	if err := s.db.View(func(t database.Transaction) error {
		var err error
		hash, err = t.FetchBlockHashByHeight(currentHeight)
		return err
	}); err != nil {
		return nil, err
	}

	msgGetBlocks := createGetBlocksMsg(hash)
	return marshalGetBlocks(msgGetBlocks)
}

func (s *synchronizer) setSyncTarget(tipHeight, maxHeight uint64) {
	s.syncTarget = tipHeight
	if tipHeight > maxHeight {
		s.syncTarget = maxHeight
	}
}

func createGetBlocksMsg(latestHash []byte) *message.GetBlocks {
	msg := &message.GetBlocks{}
	msg.Locators = append(msg.Locators, latestHash)
	return msg
}

//nolint:unparam
func marshalGetBlocks(msg *message.GetBlocks) ([]bytes.Buffer, error) {
	buf := topics.GetBlocks.ToBuffer()
	if err := msg.Encode(&buf); err != nil {
		// FIXME: shall this panic here ?  result 1 (error) is always nil (unparam)
		// log.Panic(err)
		return nil, err
	}

	return []bytes.Buffer{buf}, nil
}
