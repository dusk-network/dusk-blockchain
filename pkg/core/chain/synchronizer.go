package chain

import (
	"bytes"
	"context"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
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

	highestSeen uint64
	syncing     bool
	syncTarget  uint64
	*sequencer
	chain Ledger

	ctx context.Context

	catchBlockChan chan consensus.Results
}

// NewSynchronizer returns an initialized Synchronizer, ready for use.
func NewSynchronizer(ctx context.Context, eb eventbus.Broker, rb *rpcbus.RPCBus, chain Ledger) *Synchronizer {
	return &Synchronizer{
		eb:        eb,
		rb:        rb,
		sequencer: newSequencer(),
		ctx:       ctx,
		chain:     chain,
	}
}

// ProcessBlock handles an incoming block from the network.
func (s *Synchronizer) ProcessBlock(m message.Message) ([]bytes.Buffer, error) {
	// TODO: should the context be passed by the Peer?
	ctx := context.Background()

	blk := m.Payload().(block.Block)

	currentHeight := s.chain.CurrentHeight()

	// Is it worth looking at this?
	if blk.Header.Height <= currentHeight {
		log.Debug("discarded block from the past")
		return nil, nil
	}

	if blk.Header.Height > s.highestSeen {
		s.highestSeen = blk.Header.Height
	}

	// If this block is from far in the future, we should start syncing mode.
	if blk.Header.Height > currentHeight+1 {
		s.sequencer.add(blk)
		if !s.syncing {
			return s.startSync(blk, currentHeight)
		}

		return nil, nil
	}

	// If we are not syncing, then we should send it and forget about it.
	if !s.syncing {
		s.chain.ProcessSucceedingBlock(blk)
		return nil, nil
	}

	// Retrieve all successive blocks that need to be accepted
	blks := s.sequencer.provideSuccessors(blk)

	for _, blk := range blks {
		if err := s.chain.ProcessSyncBlock(ctx, blk); err != nil {
			log.WithError(err).Debug("could not AcceptBlock")
			return nil, err
		}

		if blk.Header.Height == s.syncTarget {
			s.syncing = false
		}
	}

	// Did we finish syncing? If so, restart the `CrunchBlocks` loop.
	if !s.syncing {
		go func() {
			if err := s.chain.CrunchBlocks(s.ctx); err != nil {
				log.WithError(err).Error("crunchBlocks exited with error")
			}
		}()
	}

	return nil, nil
}

func (s *Synchronizer) startSync(tip block.Block, currentHeight uint64) ([]bytes.Buffer, error) {
	// Kill the `CrunchBlocks` goroutine.
	select {
	case s.catchBlockChan <- consensus.Results{Blk: block.Block{}, Err: errors.New("syncing mode started")}:
	default:
	}

	s.syncTarget = tip.Header.Height
	if s.syncTarget > currentHeight+config.MaxInvBlocks {
		s.syncTarget = currentHeight + config.MaxInvBlocks
	}

	s.syncing = true

	msgGetBlocks := createGetBlocksMsg(tip.Header.Hash)
	return marshalGetBlocks(msgGetBlocks)
}

// GetSyncProgress returns how close the node is to being synced to the tip,
// as a percentage value.
func (s *Synchronizer) GetSyncProgress(ctx context.Context, e *node.EmptyRequest) (*node.SyncProgressResponse, error) {
	if s.highestSeen == 0 {
		return &node.SyncProgressResponse{Progress: 0}, nil
	}

	prevBlockHeight := s.chain.CurrentHeight()
	progressPercentage := (float64(prevBlockHeight) / float64(s.highestSeen)) * 100

	// Avoiding strange output when the chain can be ahead of the highest
	// seen block, as in most cases, consensus terminates before we see
	// the new block from other peers.
	if progressPercentage > 100 {
		progressPercentage = 100
	}

	return &node.SyncProgressResponse{Progress: float32(progressPercentage)}, nil
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
