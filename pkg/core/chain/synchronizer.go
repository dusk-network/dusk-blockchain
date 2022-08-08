// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package chain

import (
	"bytes"
	"encoding/hex"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/sirupsen/logrus"
)

const (
	syncTimeout      = time.Duration(5) * time.Second
	changeStatelabel = "change state"
)

var slog = logrus.WithField("process", "sync")

type syncState func(srcPeerAddr string, currentHeight uint64, blk block.Block, metadata *message.Metadata) ([]bytes.Buffer, error)

func (s *synchronizer) inSync(srcPeerAddr string, currentHeight uint64, blk block.Block, metadata *message.Metadata) ([]bytes.Buffer, error) {
	if blk.Header.Height > currentHeight+1 {
		s.sequencer.add(blk)

		slog.WithField("state", "outsync").Debug(changeStatelabel)

		// Trigger timeout outSync timer. If the peer initiating the Sync
		// procedure is dishonest, this timer should switch back to InSync
		// and restart Consensus Loop.
		//
		// A peer is marked as dishonest if it cannot provide a valid
		// consecutive block before the timer expires
		s.timer.Start(srcPeerAddr)

		s.state = s.outSync
		b, err := s.startSync(srcPeerAddr, blk.Header.Height, currentHeight, metadata)
		return b, err
	}

	// Otherwise notify the chain (and the consensus loop).
	if err := s.chain.TryNextConsecutiveBlockInSync(blk, metadata); err != nil {
		slog.WithField("blk_height", blk.Header.Height).
			WithField("blk_hash", hex.EncodeToString(blk.Header.Hash)).
			WithField("state", "insync").
			WithError(err).
			Warn("could not AcceptBlock")
		return nil, err
	}

	return nil, nil
}

func (s *synchronizer) outSync(srcPeerAddr string, currentHeight uint64, blk block.Block, metadata *message.Metadata) ([]bytes.Buffer, error) {
	var err error

	// Once we validate successfully the next block from the syncing
	// Peer we can consider terminating Consensus for efficiency
	// purposes.
	if currentHeight == s.hrange.from &&
		blk.Header.Height == s.hrange.from+1 {
		// This validation should happen only once to ensure we can trust
		// this peer for syncing up
		if err = s.chain.TryNextConsecutiveBlockIsValid(blk); err != nil {
			if srcPeerAddr == s.timer.ownerID {
				// Syncing Peer has provided invalid next block
				slog.WithField("r_addr", srcPeerAddr).Warn("syncing peer provided invalid next block")
				slog.WithField("state", "insync").Debug(changeStatelabel)

				s.state = s.inSync
			}

			return nil, err
		}

		log.WithField("r_addr", srcPeerAddr).Info("syncing peer provided the next block")

		s.chain.StopConsensus()
	}

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
		if err = s.chain.TryNextConsecutiveBlockOutSync(blk, metadata); err != nil {
			slog.WithError(err).WithField("state", "outsync").
				Warn("could not accept block")

			return nil, err
		}

		// Peer does provide a valid consecutive block
		// outSyncTimer should restart its counter
		if err = s.timer.Reset(srcPeerAddr); err != nil {
			slog.WithError(err).WithField("state", "outsync").
				Warn("timer error")
		}

		if blk.Header.Height == s.hrange.to {
			// Sync Target reached. outSyncTimer is not anymore needed
			s.timer.Cancel()

			// if we reach the target we get into sync mode
			// and trigger the consensus again
			if err = s.chain.RestartConsensus(); err != nil {
				return nil, err
			}

			slog.WithField("state", "insync").Debug(changeStatelabel)

			s.state = s.inSync
			break
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
	chain Ledger

	hrange struct {
		from uint64
		to   uint64
	}

	timer *outSyncTimer
}

// newSynchronizer returns an initialized synchronizer, ready for use.
func newSynchronizer(db database.DB, chain Ledger) *synchronizer {
	s := &synchronizer{
		db:        db,
		sequencer: newSequencer(),
		chain:     chain,
	}

	s.timer = newSyncTimer(syncTimeout, chain.ProcessSyncTimerExpired)

	slog.WithField("state", "insync").Debug(changeStatelabel)

	s.state = s.inSync
	return s
}

// processBlock handles an incoming block from the network.
func (s *synchronizer) processBlock(srcPeerID string, currentHeight uint64, blk block.Block, metadata *message.Metadata) (res []bytes.Buffer, err error) {
	// Clean up sequencer
	s.sequencer.cleanup(currentHeight)
	s.sequencer.dump()

	currState := s.state
	res, err = currState(srcPeerID, currentHeight, blk, metadata)
	return
}

func (s *synchronizer) startSync(strPeerAddr string, tipHeight, currentHeight uint64, _ *message.Metadata) ([]bytes.Buffer, error) {
	s.hrange.from = currentHeight
	s.setSyncTarget(tipHeight, currentHeight+config.MaxInvBlocks)

	slog.WithField("curr_h", currentHeight).
		WithField("tip", tipHeight).
		WithField("target", s.hrange.to).
		WithField("r_addr", strPeerAddr).
		Info("start syncing")

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
	s.hrange.to = tipHeight
	if tipHeight > maxHeight {
		s.hrange.to = maxHeight
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
