package consensus

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

var _ wire.Event = (*Event)(nil)

type TopicEvent struct {
	Event
	Topic topics.Topic
}

func NewTopicEvent(topic topics.Topic, hdr header.Header, payload bytes.Buffer) TopicEvent {
	return TopicEvent{
		Event{hdr, payload},
		topic,
	}
}

type Event struct {
	Header  header.Header
	Payload bytes.Buffer
}

func (e Event) Sender() []byte {
	return e.Header.Sender()
}

func (e Event) Equal(ev wire.Event) bool {
	ce, ok := ev.(Event)
	if !ok {
		return false
	}

	return e.Header.Equal(&ce.Header) && bytes.Equal(e.Payload.Bytes(), ce.Payload.Bytes())
}
