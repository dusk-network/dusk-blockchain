// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package peer

import (
	"bytes"

	log "github.com/sirupsen/logrus"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// ProcessorFunc defines an interface for callbacks which can be registered
// to the MessageProcessor, in order to process messages from the network.
type ProcessorFunc func(srcPeerID string, m message.Message) ([]bytes.Buffer, error)

// MessageProcessor is connected to all of the processing units that are tied to the peer.
// It sends an incoming message in the right direction, according to its topic.
type MessageProcessor struct {
	dupeMap    *dupemap.DupeMap
	processors map[topics.Topic]ProcessorFunc
}

// NewMessageProcessor returns an initialized MessageProcessor.
func NewMessageProcessor(bus eventbus.Broker) *MessageProcessor {
	return &MessageProcessor{
		dupeMap:    dupemap.NewDupeMapDefault(),
		processors: make(map[topics.Topic]ProcessorFunc),
	}
}

// Register a method to a certain topic. This method will be called when a message
// of the given topic is received.
func (m *MessageProcessor) Register(topic topics.Topic, fn ProcessorFunc) {
	m.processors[topic] = fn
}

// Collect a message from the network. The message is unmarshaled and passed down
// to the processing function.
func (m *MessageProcessor) Collect(srcPeerID string, packet []byte, respChan chan<- bytes.Buffer, header []byte) ([]bytes.Buffer, error) {
	b := bytes.NewBuffer(packet)

	msg, err := message.Unmarshal(b)
	if err != nil {
		return nil, err
	}

	if header != nil {
		msg = message.NewWithHeader(msg.Category(), msg.Payload(), header)
	}

	return m.process(srcPeerID, msg, respChan)
}

// CanRoute determines whether or not a message needs to be filtered by the
// dupemap.
// TODO: rename.
func (m *MessageProcessor) CanRoute(topic topics.Topic) bool {
	switch topic {
	case topics.Tx,
		topics.Candidate,
		topics.Score,
		topics.Reduction,
		topics.Agreement,
		topics.GetCandidate:
		return true
	}

	return false
}

func (m *MessageProcessor) process(srcPeerID string, msg message.Message, respChan chan<- bytes.Buffer) ([]bytes.Buffer, error) {
	category := msg.Category()
	if m.CanRoute(category) {
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
		return nil, err
	}

	if respChan != nil {
		for _, buf := range bufs {
			respChan <- buf
		}
	}

	return bufs, nil
}
