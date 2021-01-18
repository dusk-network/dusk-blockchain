// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package chain

import (
	"bytes"
	"context"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
)

// Synchronizer acts as the gateway for incoming blocks from the network.
// It decides how the Chain should process these blocks, and is responsible
// for requesting missing items in case of a desync.
type Synchronizer struct {
	eb eventbus.Broker
	rb *rpcbus.RPCBus
	db database.DB

	lock              sync.RWMutex
	highestSeenHeight uint64
	syncTargetHeight  uint64
	state             syncState

	*sequencer

	ctx context.Context

	chain Ledger
}

type syncState func(currentHeight uint64, blk block.Block) (syncState, []bytes.Buffer, error)

func (s *Synchronizer) inSync(currentHeight uint64, blk block.Block) (syncState, []bytes.Buffer, error) {
	if blk.Header.Height > currentHeight+1 {
		// If this block is from far in the future, we should start syncing mode.
		s.sequencer.add(blk)
		s.chain.StopBlockProduction()
		b, err := s.startSync(blk.Header.Height, currentHeight)
		return s.outSync, b, err
	}

	// otherwise notify the chain (and the consensus loop)
	s.chain.ProcessSucceedingBlock(blk)
	return s.inSync, nil, nil
}

func (s *Synchronizer) outSync(currentHeight uint64, blk block.Block) (syncState, []bytes.Buffer, error) {
	if blk.Header.Height > currentHeight+1 {
		// if there is a gap we add the future block to the sequencer
		s.sequencer.add(blk)
		// do we have a successive block that we might've missed?
		var err error
		blk, err = s.sequencer.get(currentHeight + 1)
		if err != nil {
			return s.outSync, nil, nil
		}
	}

	// Retrieve all successive blocks that need to be accepted
	blks := s.sequencer.provideSuccessors(blk)

	for _, blk := range blks {
		// append them all to the ledger
		if err := s.chain.ProcessSyncBlock(blk); err != nil {
			log.WithError(err).Debug("could not AcceptBlock")
			return s.outSync, nil, err
		}

		if blk.Header.Height == s.syncTarget() {
			// if we reach the target we get into sync mode
			// and trigger the consensus again
			go func() {
				if err := s.chain.ProduceBlock(); err != nil {
					// TODO we need to have a recovery procedure rather than
					// just log and forget
					log.WithError(err).Error("crunchBlocks exited with error")
				}
			}()
			return s.inSync, nil, nil
		}
	}

	return s.outSync, nil, nil
}

// NewSynchronizer returns an initialized Synchronizer, ready for use.
func NewSynchronizer(ctx context.Context, eb eventbus.Broker, rb *rpcbus.RPCBus, db database.DB, chain Ledger) *Synchronizer {
	s := &Synchronizer{
		eb:        eb,
		rb:        rb,
		db:        db,
		sequencer: newSequencer(),
		ctx:       ctx,
		chain:     chain,
	}
	s.state = s.inSync
	return s
}

// ProcessBlock handles an incoming block from the network.
func (s *Synchronizer) ProcessBlock(m message.Message) (res []bytes.Buffer, err error) {
	blk := m.Payload().(block.Block)

	currentHeight := s.chain.CurrentHeight()

	// Is it worth looking at this?
	if blk.Header.Height <= currentHeight {
		log.Debug("discarded block from the past")
		return
	}

	s.lock.Lock()
	currState := s.state
	if blk.Header.Height > s.highestSeenHeight {
		s.highestSeenHeight = blk.Header.Height
	}
	s.lock.Unlock()

	var newState syncState
	newState, res, err = currState(currentHeight, blk)
	s.setState(newState)

	return
}

func (s *Synchronizer) startSync(tipHeight, currentHeight uint64) ([]bytes.Buffer, error) {
	s.setSyncTarget(tipHeight, currentHeight+config.MaxInvBlocks)

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

// GetSyncProgress returns how close the node is to being synced to the tip,
// as a percentage value.
func (s *Synchronizer) GetSyncProgress(ctx context.Context, e *node.EmptyRequest) (*node.SyncProgressResponse, error) {
	if s.highestSeen() == 0 {
		return &node.SyncProgressResponse{Progress: 0}, nil
	}

	prevBlockHeight := s.chain.CurrentHeight()
	progressPercentage := (float64(prevBlockHeight) / float64(s.highestSeen())) * 100

	// Avoiding strange output when the chain can be ahead of the highest
	// seen block, as in most cases, consensus terminates before we see
	// the new block from other peers.
	if progressPercentage > 100 {
		progressPercentage = 100
	}

	return &node.SyncProgressResponse{Progress: float32(progressPercentage)}, nil
}

// highestSeen gets highestSeenHeight safely
func (s *Synchronizer) highestSeen() uint64 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.highestSeenHeight
}

func (s *Synchronizer) setState(newState syncState) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.state = newState
}

// syncTarget gets syncTargetHeight safely
func (s *Synchronizer) syncTarget() uint64 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.syncTargetHeight
}
func (s *Synchronizer) setSyncTarget(tipHeight, maxHeight uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.syncTargetHeight = tipHeight
	if s.syncTargetHeight > maxHeight {
		s.syncTargetHeight = maxHeight
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
		//FIXME: shall this panic here ?  result 1 (error) is always nil (unparam)
		//log.Panic(err)
		return nil, err
	}

	return []bytes.Buffer{buf}, nil
}
