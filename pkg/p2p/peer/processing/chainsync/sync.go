package chainsync

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/peermsg"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/v2/block"
	logger "github.com/sirupsen/logrus"
)

var log *logger.Entry = logger.WithFields(logger.Fields{"process": "synchronizer"})

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

	// Notify `Chain` of our highest seen block
	if s.getHighestSeen() < height {
		s.setHighestSeen(height)
		s.publishHighestSeen(height)
	}

	blk, err := s.getLastBlock()
	if err != nil {
		return err
	}

	log.WithField("our height", blk.Header.Height).WithField("received block height", height).Debugln("block received")
	// Only ask for missing blocks if we are not currently syncing, to prevent
	// asking many peers for (generally) the same blocks.
	diff := compareHeights(blk.Header.Height, height)
	if !s.IsSyncing() && diff > 1 {

		hash := base64.StdEncoding.EncodeToString(blk.Header.Hash)
		log.Debugf("Start syncing from %s", peerInfo)
		log.Debugf("Local tip: height %d [%s]", blk.Header.Height, hash)

		msg := createGetBlocksMsg(blk.Header.Hash)
		buf, err := marshalGetBlocks(msg)
		if err != nil {
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
			return err
		}

		if err := message.UnmarshalBlock(buf, &blk); err != nil {
			return err
		}

		msg := message.New(topics.Block, *blk)
		s.publisher.Publish(topics.Block, msg)
	}

	return nil
}

func (s *ChainSynchronizer) getLastBlock() (block.Block, error) {
	req := rpcbus.NewRequest(nil)
	resp, err := s.rpcBus.Call(topics.GetLastBlock, req, 2*time.Second)
	if err != nil {
		return block.Block{}, err
	}
	blk := resp.(block.Block)
	return blk, nil
}

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

// TODO: interface - get rid of the marshalling
func (s *ChainSynchronizer) publishHighestSeen(height uint64) {
	msg := message.New(topics.HighestSeen, height)
	s.publisher.Publish(topics.HighestSeen, msg)
}

func compareHeights(ourHeight, theirHeight uint64) int64 {
	return int64(theirHeight) - int64(ourHeight)
}

func createGetBlocksMsg(latestHash []byte) *peermsg.GetBlocks {
	msg := &peermsg.GetBlocks{}
	msg.Locators = append(msg.Locators, latestHash)
	return msg
}

func marshalGetBlocks(msg *peermsg.GetBlocks) (*bytes.Buffer, error) {
	buf := topics.GetBlocks.ToBuffer()
	if err := msg.Encode(&buf); err != nil {
		log.Panic(err)
	}

	return &buf, nil
}

func peekBlockHeight(r *bufio.Reader) (uint64, error) {
	// The block height is a little-endian uint64, starting at index 1 of the buffer
	// It is preceded by the version (1 byte)
	bytes, err := r.Peek(9)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(bytes[1:9]), nil
}
