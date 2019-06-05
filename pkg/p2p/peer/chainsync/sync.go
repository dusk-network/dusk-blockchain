package chainsync

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"net"
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func LaunchChainSynchronizer(eventBroker wire.EventBroker, magic protocol.Magic) *ChainSynchronizer {
	cs := newChainSynchronizer(eventBroker, magic)
	go cs.listen()
	return cs
}

type Synchronizer interface {
	Synchronize(net.Conn, <-chan *bytes.Buffer)
}

type ChainSynchronizer struct {
	publisher         wire.EventPublisher
	lock              sync.RWMutex
	latestHeader      *block.Header
	gossip            *processing.Gossip
	acceptedBlockChan <-chan block.Block
}

func newChainSynchronizer(eventBroker wire.EventBroker, magic protocol.Magic) *ChainSynchronizer {
	return &ChainSynchronizer{
		publisher: eventBroker,
		gossip:    processing.NewGossip(magic),
		// acceptedBlockChan: consensus.InitAcceptedBlockUpdate(eventBroker),
	}
}

func (s *ChainSynchronizer) Synchronize(conn net.Conn, blockChan <-chan *bytes.Buffer) {
	for {
		blkBuf := <-blockChan
		if s.isNext(blkBuf) {
			s.publisher.Publish(string(topics.Block), blkBuf)
			continue
		}

		s.askForMissingBlocks(conn)
	}
}

func (s *ChainSynchronizer) listen() {
	for {
		blk := <-s.acceptedBlockChan
		s.lock.Lock()
		s.latestHeader = blk.Header
		s.lock.Unlock()
	}
}

func (s *ChainSynchronizer) askForMissingBlocks(conn net.Conn) error {
	msg := createGetBlocksMsg(s.getCurrentHash())
	return s.sendGetBlocksMsg(msg, conn)
}

func (s *ChainSynchronizer) isNext(b *bytes.Buffer) bool {
	height := peekBlockHeight(b)
	currentHeight := s.getCurrentHeight()
	return height-currentHeight <= 1
}

func createGetBlocksMsg(latestHash []byte) *peermsg.GetBlocks {
	msg := &peermsg.GetBlocks{}
	msg.Locators = append(msg.Locators, latestHash)
	// Set the target to a zero value, so we get as many blocks as possible.
	msg.Target = make([]byte, 32)
	return msg
}

func (s *ChainSynchronizer) sendGetBlocksMsg(msg *peermsg.GetBlocks, conn net.Conn) error {
	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		panic(err)
	}

	bufWithTopic, err := wire.AddTopic(buf, topics.GetBlocks)
	if err != nil {
		return err
	}

	encodedMsg, err := s.gossip.Process(bufWithTopic)
	if err != nil {
		return err
	}

	_, err = conn.Write(encodedMsg.Bytes())
	return err
}

func peekBlockHeight(buf *bytes.Buffer) uint64 {
	r := bufio.NewReader(buf)

	// The block height is a little-endian uint64, starting at index 9 of the buffer
	// It is preceded by the version (1 byte) and the timestamp (8 bytes)
	bytes, err := r.Peek(17)
	if err != nil {
		panic(err)
	}

	return binary.LittleEndian.Uint64(bytes[9:17])
}

func (s *ChainSynchronizer) getCurrentHeight() uint64 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.latestHeader.Height
}

func (s *ChainSynchronizer) getCurrentHash() []byte {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.latestHeader.Hash
}
