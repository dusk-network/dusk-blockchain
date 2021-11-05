// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package peer

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// ProcessorFunc defines an interface for callbacks which can be registered
// to the MessageProcessor, in order to process messages from the network.
type ProcessorFunc func(srcPeerID string, m message.Message) ([]bytes.Buffer, error)

// MessageProcessor is connected to all of the processing units that are tied to the peer.
// It sends an incoming message in the right direction, according to its topic.
type MessageProcessor struct {
	dupeMap, repropagateMap *dupemap.DupeMap
	processors              map[topics.Topic]ProcessorFunc
	bus                     eventbus.Broker
}

// NewMessageProcessor returns an initialized MessageProcessor.
func NewMessageProcessor(bus eventbus.Broker) *MessageProcessor {
	return &MessageProcessor{
		dupeMap:        dupemap.NewDupeMapDefault(),
		repropagateMap: dupemap.NewDupeMapDefault(),
		processors:     make(map[topics.Topic]ProcessorFunc),
		bus:            bus,
	}
}

// Register a method to a certain topic. This method will be called when a message
// of the given topic is received.
func (m *MessageProcessor) Register(topic topics.Topic, fn ProcessorFunc) {
	m.processors[topic] = fn
}

// Collect a message from the network. The message is unmarshaled and passed down
// to the processing function.
func (m *MessageProcessor) Collect(srcPeerID string, packet []byte, respRingBuf *ring.Buffer, services protocol.ServiceFlag, header []byte) ([]bytes.Buffer, error) {
	if len(packet) == 0 {
		return nil, errors.New("empty packet provided")
	}
	defer m.trace("collected", srcPeerID, time.Now().UnixNano(), packet)

	b := bytes.NewBuffer(packet)
	topic := topics.Topic(b.Bytes()[0])

	msg, err := message.Unmarshal(b, header)
	if err != nil {
		return nil, fmt.Errorf("error while unmarshaling: %s - topic: %s", err, topic)
	}

	return m.process(srcPeerID, msg, respRingBuf, services)
}

func (m *MessageProcessor) trace(tag, srcAddr string, st int64, msg []byte) {
	if l.Logger.GetLevel() == log.TraceLevel {
		duration := float64(time.Now().UnixNano()-st) / 1000000

		var topicName string
		if len(msg) > 0 {
			topicName = topics.Topic(msg[0]).String()
		}

		l.WithField("process", "msg_processor").
			WithField("len", len(msg)).
			WithField("ms", duration).
			WithField("topic", topicName).
			WithField("src", srcAddr).
			WithField("tag", tag).
			Trace("wire msg")
	}
}

// shouldBeCached determines types of topics that are filtered with a dupemap.
// topics.Inv is filtered with another instance of dupemap managed by responding.DataRequestor.
func (m *MessageProcessor) shouldBeCached(t topics.Topic) bool {
	switch t {
	case topics.Tx,
		topics.Candidate,
		topics.NewBlock,
		topics.Reduction,
		topics.Agreement,
		topics.GetCandidate:
		return true
	default:
		return false
	}
}

// Repropagate basic consensus messages.
func (m *MessageProcessor) Repropagate(srcPeerID string, packet []byte, services protocol.ServiceFlag) error {
	if len(packet) == 0 {
		return errors.New("empty packet provided")
	}

	b := bytes.NewBuffer(packet)
	topic := topics.Topic(b.Bytes()[0])

	msg, err := message.Unmarshal(b, nil)
	if err != nil {
		return fmt.Errorf("error while unmarshaling: %s - topic: %s", err, topic)
	}

	// Repropagate
	category := msg.Category()
	switch category {
	case
		topics.NewBlock,
		topics.Reduction,
		topics.Agreement:
	default:
		return nil
	}

	if !m.repropagateMap.HasAnywhere(bytes.NewBuffer(msg.Id())) {
		return nil
	}

	buf, err := message.Marshal(msg)
	if err != nil {
		return err
	}

	serialized := message.New(msg.Category(), buf)
	m.bus.Publish(topics.Gossip, serialized)

	return nil
}

func (m *MessageProcessor) process(srcPeerID string, msg message.Message, respRingBuf *ring.Buffer, services protocol.ServiceFlag) ([]bytes.Buffer, error) {
	category := msg.Category()
	if !canRoute(services, category) {
		return nil, fmt.Errorf("attempted to process an illegal topic %s for node type %v", category, services)
	}

	if m.shouldBeCached(category) {
		if !m.dupeMap.HasAnywhere(bytes.NewBuffer(msg.Id())) {
			return nil, nil
		}
	}

	processFn, ok := m.processors[category]
	if !ok {
		log.WithField("topic", category).Debugln("received message with unknown topic")
		return nil, nil
	}

	bufs, err := processFn(srcPeerID, msg)
	if err != nil {
		return nil, fmt.Errorf("error while processing: %s - topic %s", err, msg.Category())
	}

	if respRingBuf != nil {
		for _, buf := range bufs {
			e := ring.Elem{Data: buf.Bytes()}
			if !respRingBuf.Put(e) {
				log.WithError(err).Errorln("could not send response")
				return nil, err
			}
		}
	}

	return bufs, nil
}
