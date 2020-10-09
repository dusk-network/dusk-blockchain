package chainsync

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"

	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/peermsg"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	logger "github.com/sirupsen/logrus"
)

var log = logger.WithFields(logger.Fields{"process": "sync"})

// ChainSynchronizer is the component responsible for keeping the node in sync with the
// rest of the network. It sits between the peer and the chain, as a sort of gateway for
// incoming blocks. It keeps track of the local chain tip and compares it with each incoming
// block to make sure we stay in sync with our peers.
type ChainSynchronizer struct {
	publisher eventbus.Publisher
	rpcBus    *rpcbus.RPCBus
	*Counter
	responseChan chan<- *bytes.Buffer

	// Highest block we've seen. We keep track of it, so that we do not
	// spam the `Chain` with messages during a sync.
	lock        sync.RWMutex
	highestSeen uint64
}

// NewChainSynchronizer returns an initialized ChainSynchronizer. The passed responseChan
// should point to an individual peer's outgoing message queue, and the passed Counter
// should be shared between all instances of the ChainSynchronizer.
func NewChainSynchronizer(publisher eventbus.Publisher, rpcBus *rpcbus.RPCBus, responseChan chan<- *bytes.Buffer, counter *Counter) *ChainSynchronizer {
	return &ChainSynchronizer{
		publisher:    publisher,
		rpcBus:       rpcBus,
		Counter:      counter,
		responseChan: responseChan,
	}
}

// HandleBlock mainly busy with two activities
// Unmarshal and republish block or
// Switch to Syncing mode by triggering syncing procedure
func (s *ChainSynchronizer) HandleBlock(blkBuf *bytes.Buffer, sendingPeerAddr string) error {

	// wire message of type topics.Block
	// sending peer address
	fields := logger.Fields{
		"sender": sendingPeerAddr,
	}

	msg := bufio.NewReader(blkBuf)
	recvBlockHeight, err := peekBlockHeight(msg)
	if err != nil {
		log.WithFields(fields).WithError(err).Warn("failed to read block height")
		return err
	}

	// received block height (from the message but not from the unmarshaled block)
	fields["recv_blk_h"] = recvBlockHeight

	// Notify `Chain` of our highest seen block
	// TODO: Chain processing should not be disturbed with highestSeen req
	if s.getHighestSeen() < recvBlockHeight {
		s.setHighestSeen(recvBlockHeight)
		s.publishHighestSeen(recvBlockHeight)
	}

	// Request blockchain last block
	lastBlk, err := s.getLastBlock()
	if err != nil {
		log.WithFields(fields).WithError(err).Error("failed to getLastBlock")
		return err
	}
	diff := int64(recvBlockHeight - lastBlk.Header.Height)

	fields["local_h"] = lastBlk.Header.Height
	log.WithFields(fields).Debug("process topics.Block")

	// ChainSynchronizer supports sync mode and non-sync mode
	// Handle the wire block message accordingly
	var resultErr error
	if s.IsSyncing() {
		resultErr = s.handleSyncingMode(sendingPeerAddr, msg, diff, fields)
	} else {
		resultErr = s.handleNonSyncingMode(sendingPeerAddr, msg, diff, lastBlk, fields)
	}

	return resultErr
}

func (s *ChainSynchronizer) handleNonSyncingMode(sendingPeerAddr string, msg io.Reader, diff int64, lastBlk block.Block, fields logger.Fields) error {

	// Difference between local height and sending peer height is more than 1 block.
	if diff > 1 {

		// Switch to Syncing Mode
		// Trigger syncing procedure with this peer for fetching
		// up to maxBlocksCount blocks.

		// marshal wire message topics.GetBlocks
		msgGetBlocks := createGetBlocksMsg(lastBlk.Header.Hash)
		buf, err := marshalGetBlocks(msgGetBlocks)
		if err != nil {
			log.WithFields(fields).Error("could not marshalGetBlocks")
			return err
		}

		// The peer who first acquires sync lock state can trigger syncing procedure
		if syncingPeer, err := s.StartSyncing(uint64(diff), sendingPeerAddr); err != nil {
			// already started syncing with another peer
			log.WithFields(fields).WithField("syncing_peer", syncingPeer).Warn("failed to start syncing")
			return nil
		}

		log.WithFields(fields).
			WithField("last_blk_hash", hex.EncodeToString(lastBlk.Header.Hash)).
			WithField("diff", diff).
			Info("start syncing")

		// Respond back to the sending Peer with topics.GetBlocks wire message
		s.responseChan <- buf

	} else {

		// Difference between local height and sending peer height is 1 block.
		// No need for resyncing, just republish this block to the upper layers
		if diff == 1 {

			log.WithFields(fields).Info("republish a block")
			if err := s.republish(msg); err != nil {
				log.WithFields(fields).WithError(err).Warn("failed to republish")
				return err
			}
		}
	}

	return nil
}

func (s *ChainSynchronizer) handleSyncingMode(sendingPeerAddr string, r io.Reader, diff int64, fields logger.Fields) error {

	syncPeerAddr := s.GetSyncingPeer()

	// Republish blocks only from the syncing peer while in syncing mode
	if len(syncPeerAddr) > 0 && (syncPeerAddr == sendingPeerAddr) && diff > 0 {

		log.WithFields(fields).Info("republish a block from syncing peer")
		if err := s.republish(r); err != nil {
			log.WithFields(fields).WithError(err).Warn("failed to republish")
			return err
		}

	} else {

		if diff == 1 {
			// While we are in syncing mode, another peer is sending us the next block
			// That might mean be an indication that current syncingPeer is cheating
			// TODO: detect if syncing peer misbehaving. If so, blacklist it
			log.WithFields(fields).Warn("republish a block from a non-syncing peer")

			if err := s.republish(r); err != nil {
				log.WithFields(fields).WithError(err).Warn("failed to republish")
				return err
			}

		} else {
			log.WithFields(fields).WithField("syncPeer", syncPeerAddr).
				Debug("Skip block, still syncing with another peer")
		}
	}

	return nil
}

// republish unmarshals a block from a Reader and republish it
func (s *ChainSynchronizer) republish(r io.Reader) error {

	// Write bufio.Reader into a bytes.Buffer and unmarshal it so we can send it over the event bus.
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(r); err != nil {
		return err
	}

	blk := block.NewBlock()
	if err := message.UnmarshalBlock(buf, blk); err != nil {
		return err
	}

	msg := message.New(topics.Block, *blk)
	errList := s.publisher.Publish(topics.Block, msg)
	diagnostics.LogPublishErrors("chainsync/sync.go, topics.Block", errList)

	return nil
}

func (s *ChainSynchronizer) getLastBlock() (block.Block, error) {
	req := rpcbus.NewRequest(nil)
	timeoutGetLastBlock := time.Duration(config.Get().Timeout.TimeoutGetLastBlock) * time.Second
	resp, err := s.rpcBus.Call(topics.GetLastBlock, req, timeoutGetLastBlock)
	if err != nil {
		log.
			WithError(err).
			Error("timeout topics.GetLastBlock")
		return block.Block{}, err
	}
	blk := resp.(block.Block)
	return blk, nil
}

//  TODO: To be moved out of ChainSynchronizer
func (s *ChainSynchronizer) getHighestSeen() uint64 {
	s.lock.RLock()
	height := s.highestSeen
	s.lock.RUnlock()
	return height
}

func (s *ChainSynchronizer) setHighestSeen(height uint64) {
	s.lock.Lock()
	s.highestSeen = height
	s.lock.Unlock()
}

func (s *ChainSynchronizer) publishHighestSeen(height uint64) {
	msg := message.New(topics.HighestSeen, height)
	errList := s.publisher.Publish(topics.HighestSeen, msg)
	diagnostics.LogPublishErrors("", errList)
}

func createGetBlocksMsg(latestHash []byte) *peermsg.GetBlocks {
	msg := &peermsg.GetBlocks{}
	msg.Locators = append(msg.Locators, latestHash)
	return msg
}

//nolint:unparam
func marshalGetBlocks(msg *peermsg.GetBlocks) (*bytes.Buffer, error) {
	buf := topics.GetBlocks.ToBuffer()
	if err := msg.Encode(&buf); err != nil {
		//FIXME: shall this panic here ?  result 1 (error) is always nil (unparam)
		//log.Panic(err)
		return nil, err
	}

	return &buf, nil
}

func peekBlockHeight(r *bufio.Reader) (uint64, error) {
	// The block height is a little-endian uint64, starting at index 1 of the buffer
	// It is preceded by the version (1 byte)
	b, err := r.Peek(9)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(b[1:9]), nil
}
