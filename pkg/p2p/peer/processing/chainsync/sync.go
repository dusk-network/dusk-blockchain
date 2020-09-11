package chainsync

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
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

var log = logger.WithFields(logger.Fields{"process": "synchronizer"})

// ChainSynchronizer is the component responsible for keeping the node in sync with the
// rest of the network. It sits between the peer and the chain, as a sort of gateway for
// incoming blocks. It keeps track of the local chain tip and compares it with each incoming
// block to make sure we stay in sync with our peers.
type ChainSynchronizer struct {
	publisher eventbus.Publisher
	rpcBus    *rpcbus.RPCBus
	*Counter
	responseChan chan<- *bytes.Buffer

	lock sync.RWMutex
	// Highest block we've seen. We keep track of it, so that we do not
	// spam the `Chain` with messages during a sync.
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

// Synchronize our blockchain with our peers.
func (s *ChainSynchronizer) Synchronize(blkBuf *bytes.Buffer, peerInfo string) error {
	r := bufio.NewReader(blkBuf)
	height, err := peekBlockHeight(r)
	if err != nil {
		return err
	}
	log.WithField("height", height).Debug("Synchronize")

	// Notify `Chain` of our highest seen block
	if s.getHighestSeen() < height {
		s.setHighestSeen(height)
		s.publishHighestSeen(height)
	}

	lastBlk, err := s.getLastBlock()
	if err != nil {
		log.
			WithError(err).
			WithField("height", height).Error("failed to getLastBlock")
		return err
	}

	log.WithField("last_height", lastBlk.Header.Height).WithField("height", height).Debugln("block received")
	// Only ask for missing blocks if we are not currently syncing, to prevent
	// asking many peers for (generally) the same blocks.
	diff := compareHeights(lastBlk.Header.Height, height)
	if !s.IsSyncing() && diff > 1 {

		hash := base64.StdEncoding.EncodeToString(lastBlk.Header.Hash)
		log.
			WithField("height", lastBlk.Header.Height).
			WithField("hash", hash).
			WithField("peerInfo", peerInfo).
			Debug("Start syncing")

		msg := createGetBlocksMsg(lastBlk.Header.Hash)
		buf, err := marshalGetBlocks(msg)
		if err != nil {
			log.
				WithField("height", lastBlk.Header.Height).
				WithField("hash", hash).
				WithField("peerInfo", peerInfo).
				Error("could not marshalGetBlocks")
			return err
		}

		s.responseChan <- buf
		s.StartSyncing(uint64(diff))
		return nil
	}

	// Does the block come directly after our most recent one?
	if diff == 1 {
		// Write bufio.Reader into a bytes.Buffer and unmarshal it so we can send it over the event bus.
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(r); err != nil {
			log.
				WithError(err).
				WithField("height", height).Error("failed to ReadFrom")
			return err
		}

		blk := block.NewBlock()
		if err := message.UnmarshalBlock(buf, blk); err != nil {
			log.
				WithError(err).
				WithField("height", height).Error("failed to UnmarshalBlock")
			return err
		}

		msg := message.New(topics.Block, *blk)
		errList := s.publisher.Publish(topics.Block, msg)
		diagnostics.LogPublishErrors("chainsync/sync.go, topics.Block", errList)

	}

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

func (s *ChainSynchronizer) getHighestSeen() uint64 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	height := s.highestSeen
	return height
}

func (s *ChainSynchronizer) setHighestSeen(height uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.highestSeen = height
}

func (s *ChainSynchronizer) publishHighestSeen(height uint64) {
	msg := message.New(topics.HighestSeen, height)
	errList := s.publisher.Publish(topics.HighestSeen, msg)
	diagnostics.LogPublishErrors("", errList)
}

func compareHeights(ourHeight, theirHeight uint64) int64 {
	return int64(theirHeight) - int64(ourHeight)
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
