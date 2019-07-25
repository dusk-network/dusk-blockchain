package chainsync

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"

	logger "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

var log *logger.Entry = logger.WithFields(logger.Fields{"process": "synchronizer"})

// ChainSynchronizer is the component responsible for keeping the node in sync with the
// rest of the network. It sits between the peer and the chain, as a sort of gateway for
// incoming blocks. It keeps track of the local chain tip and compares it with each incoming
// block to make sure we stay in sync with our peers.
type ChainSynchronizer struct {
	publisher wire.EventPublisher
	rpcBus    *wire.RPCBus
	*Counter
	responseChan chan<- *bytes.Buffer
}

// NewChainSynchronizer returns an initialized ChainSynchronizer. The passed responseChan
// should point to an individual peer's outgoing message queue, and the passed Counter
// should be shared between all instances of the ChainSynchronizer.
func NewChainSynchronizer(publisher wire.EventPublisher, rpcBus *wire.RPCBus, responseChan chan<- *bytes.Buffer, counter *Counter) *ChainSynchronizer {
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

	blk, err := s.getLastBlock()
	if err != nil {
		return err
	}

	log.WithField("our height", blk.Header.Height).WithField("received block height", height).Debugln("block received")
	// Only ask for missing blocks if we are not currently syncing, to prevent
	// asking many peers for (generally) the same blocks.
	diff := compareHeights(blk.Header.Height, height)
	if !s.isSyncing() && diff > 1 {

		hash := base64.StdEncoding.EncodeToString(blk.Header.Hash)
		log.Debugf("Start syncing from %s", peerInfo)
		log.Debugf("Local tip: height %d [%s]", blk.Header.Height, hash)

		msg := createGetBlocksMsg(blk.Header.Hash)
		buf, err := marshalGetBlocks(msg)
		if err != nil {
			return err
		}

		s.responseChan <- buf
		s.startSyncing(uint64(diff))
		return nil
	}

	if diff == 1 {
		// Write bufio.Reader into a bytes.Buffer so we can send it over the event bus.
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(r); err != nil {
			return err
		}

		s.publisher.Publish(string(topics.Block), buf)
	}

	return nil
}

func (s *ChainSynchronizer) getLastBlock() (*block.Block, error) {
	req := wire.NewRequest(bytes.Buffer{}, 2)
	blkBuf, err := s.rpcBus.Call(wire.GetLastBlock, req)
	if err != nil {
		return nil, err
	}

	blk := block.NewBlock()
	if err := blk.Decode(&blkBuf); err != nil {
		return nil, err
	}

	return blk, nil
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
	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		panic(err)
	}

	return wire.AddTopic(buf, topics.GetBlocks)
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
