package chainsync

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
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
	target            []byte
}

func newChainSynchronizer(eventBroker wire.EventBroker, magic protocol.Magic) *ChainSynchronizer {
	return &ChainSynchronizer{
		publisher:         eventBroker,
		gossip:            processing.NewGossip(magic),
		acceptedBlockChan: consensus.InitAcceptedBlockUpdate(eventBroker),
		target:            make([]byte, 32),
	}
}

func (s *ChainSynchronizer) Synchronize(conn net.Conn, blockChan <-chan *bytes.Buffer) {
	for {
		blkBuf := <-blockChan
		r := bufio.NewReader(blkBuf)
		if s.isNext(r) {
			s.publisher.Publish(string(topics.Block), blkBuf)
			continue
		}

		// Only ask for missing blocks if we don't have a current sync target
		if s.noTarget() {
			if err := s.askForMissingBlocks(conn, r); err != nil {
				fmt.Println(err)
			}
		}
	}
}

func (s *ChainSynchronizer) listen() {
	for {
		blk := <-s.acceptedBlockChan
		s.setLatestHeader(blk.Header)
		// Empty our sync target if we receive the wanted block
		if bytes.Equal(s.target, blk.Header.Hash) {
			s.eraseTarget()
		}
	}
}

func (s *ChainSynchronizer) askForMissingBlocks(conn net.Conn, r io.Reader) error {
	blk := block.NewBlock()
	if err := blk.Decode(r); err != nil {
		return err
	}
	s.setTarget(blk.Header.Hash)
	msg := createGetBlocksMsg(s.currentHash(), blk.Header.Hash)
	return s.sendGetBlocksMsg(msg, conn)
}

func (s *ChainSynchronizer) isNext(r *bufio.Reader) bool {
	height := peekBlockHeight(r)
	currentHeight := s.currentHeight()
	return height-currentHeight <= 1
}

func createGetBlocksMsg(latestHash, target []byte) *peermsg.GetBlocks {
	msg := &peermsg.GetBlocks{}
	msg.Locators = append(msg.Locators, latestHash)
	// Set the target to a zero value, so we get as many blocks as possible.
	msg.Target = target
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

func peekBlockHeight(r *bufio.Reader) uint64 {
	// The block height is a little-endian uint64, starting at index 1 of the buffer
	// It is preceded by the version (1 byte)
	bytes, err := r.Peek(9)
	if err != nil {
		panic(err)
	}

	return binary.LittleEndian.Uint64(bytes[1:9])
}

func (s *ChainSynchronizer) setLatestHeader(header *block.Header) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.latestHeader = header
}

func (s *ChainSynchronizer) setTarget(target []byte) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.target = target
}

func (s *ChainSynchronizer) eraseTarget() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.target = make([]byte, 32)
}
func (s *ChainSynchronizer) currentHeight() uint64 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.latestHeader.Height
}

func (s *ChainSynchronizer) currentHash() []byte {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.latestHeader.Hash
}

func (s *ChainSynchronizer) noTarget() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return bytes.Equal(s.target, make([]byte, 32))
}
