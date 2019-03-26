package peermgr

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// LaunchGossipCollectors will launch a gossip collector for every relevant
// topic.
func LaunchGossipCollectors(eventBus *wire.EventBus, magic protocol.Magic) {
	initGossipCollector(eventBus, string(topics.GossipCandidate),
		topics.Candidate, magic)
	initGossipCollector(eventBus, string(topics.GossipScore),
		topics.Score, magic)
	initGossipCollector(eventBus, string(topics.GossipSigSet),
		topics.SigSet, magic)
	initGossipCollector(eventBus, string(topics.GossipBlockReduction),
		topics.BlockReduction, magic)
	initGossipCollector(eventBus, string(topics.GossipSigSetReduction),
		topics.SigSetReduction, magic)
	initGossipCollector(eventBus, string(topics.GossipBlockAgreement),
		topics.BlockAgreement, magic)
	initGossipCollector(eventBus, string(topics.GossipSigSetAgreement),
		topics.SigSetAgreement, magic)
}

type gossipCollector struct {
	eventBus    *wire.EventBus
	headerTopic topics.Topic
	magic       protocol.Magic
}

func initGossipCollector(eventBus *wire.EventBus, topic string,
	headerTopic topics.Topic, magic protocol.Magic) {

	collector := &gossipCollector{
		eventBus:    eventBus,
		headerTopic: headerTopic,
		magic:       magic,
	}
	go wire.NewEventSubscriber(eventBus, collector, topic).Accept()
}

func (gc *gossipCollector) Collect(message *bytes.Buffer) error {
	messageWithHeader, err := gc.addHeader(message)
	if err != nil {
		return err
	}

	gc.eventBus.Publish(string(topics.Propagate), messageWithHeader)
	return nil
}

func (gc *gossipCollector) addHeader(message *bytes.Buffer) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	if err := encoding.WriteUint32(buffer, binary.LittleEndian, uint32(gc.magic)); err != nil {
		return nil, err
	}

	gc.writeTopic(buffer)

	payloadLength := uint32(message.Len())
	if err := encoding.WriteUint32(buffer, binary.LittleEndian, payloadLength); err != nil {
		return nil, err
	}

	checksum, err := crypto.Checksum(message.Bytes())
	if err != nil {
		return nil, err
	}
	if err := encoding.WriteUint32(buffer, binary.LittleEndian, checksum); err != nil {
		return nil, err
	}

	if _, err := buffer.Write(message.Bytes()); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (gc *gossipCollector) writeTopic(r *bytes.Buffer) error {
	topicBytes := topics.TopicToByteArray(gc.headerTopic)
	if _, err := r.Write(topicBytes[:]); err != nil {
		return err
	}

	return nil
}
