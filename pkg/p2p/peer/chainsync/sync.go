package chainsync

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"net"
	"sync"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func LaunchChainSynchronizer(eventBroker wire.EventBroker, magic protocol.Magic) *ChainSynchronizer {
	return newChainSynchronizer(eventBroker, magic)
}

type Synchronizer interface {
	Synchronize(net.Conn, <-chan *bytes.Buffer)
}

type ChainSynchronizer struct {
	publisher    wire.EventPublisher
	lock         sync.RWMutex
	latestHeader *block.Header
	gossip       *processing.Gossip
	target       *block.Block
}

func newChainSynchronizer(eventBroker wire.EventBroker, magic protocol.Magic) *ChainSynchronizer {
	cs := &ChainSynchronizer{
		publisher: eventBroker,
		gossip:    processing.NewGossip(magic),
	}

	eventBroker.SubscribeCallback(string(topics.AcceptedBlock), cs.updateHeader)
	return cs
}

func (s *ChainSynchronizer) Synchronize(conn net.Conn, blockChan <-chan *bytes.Buffer) {
	for {
		blkBuf := <-blockChan
		r := bufio.NewReader(blkBuf)
		height := peekBlockHeight(r)

		// Only ask for missing blocks if we don't have a current sync target
		if s.noTarget() && s.amBehind(height) {
			blk := block.NewBlock()
			if err := blk.Decode(r); err != nil {
				return err
			}
			s.setTarget(blk)
			s.askForMissingBlocks(conn, blk)
			continue
		}

		if err := s.publishBlock(r); err != nil {
			log.WithFields(log.Fields{
				"process": "synchronizer",
				"error":   err,
			}).Errorln("problem publishing block")
		}

		if height == s.targetHeight()-1 {
			if err := s.publishTarget(); err != nil {
				log.WithFields(log.Fields{
					"process": "synchronizer",
					"error":   err,
				}).Errorln("problem publishing target block")
			}
			s.eraseTarget()
		}
	}
}

func (s *ChainSynchronizer) publishBlock(r *bufio.Reader) error {
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(r); err != nil {
		return err
	}
	s.publisher.Publish(string(topics.Block), buf)
	return nil
}

func (s *ChainSynchronizer) publishTarget() error {
	targetBuf := new(bytes.Buffer)
	if err := s.target.Encode(targetBuf); err != nil {
		return err
	}
	s.publisher.Publish(string(topics.Block), targetBuf)
	return nil
}

func (s *ChainSynchronizer) updateHeader(m *bytes.Buffer) error {
	blk := block.NewBlock()
	if err := blk.Decode(m); err != nil {
		return err
	}

	s.setLatestHeader(blk.Header)
	return nil
}

func (s *ChainSynchronizer) askForMissingBlocks(conn net.Conn, blk *block.Block) {
	msg := createGetBlocksMsg(s.currentHash(), blk.Header.Hash)
	return s.sendGetBlocksMsg(msg, conn)
}

func (s *ChainSynchronizer) amBehind(height uint64) bool {
	currentHeight := s.currentHeight()
	return int64(height)-int64(currentHeight) > 1
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

func (s *ChainSynchronizer) setTarget(target *block.Block) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.target = target
}

func (s *ChainSynchronizer) eraseTarget() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.target = nil
}

func (s *ChainSynchronizer) targetHeight() uint64 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.target != nil {
		return s.target.Header.Height
	}
	return 0
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
	return s.target == nil
}
