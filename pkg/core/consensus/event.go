package consensus

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

var _ wire.Event = (*Event)(nil)

// TopicEvent is the concatenation of a consensus Event, and it's designated Topic.
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

// Event is the collection of a consensus message header and it's payload.
// Its primary purpose is to group all of the common fields in a consensus message
// together, and allow for consensus components to process the topic-specific payload
// on its own, while retaining the general information if needed.
type Event struct {
	Header  header.Header
	Payload bytes.Buffer
}

// Sender returns the BLS public key of the event sender.
func (e Event) Sender() []byte {
	return e.Header.Sender()
}

// Equal checks if an Event is equal to another.
func (e Event) Equal(ev wire.Event) bool {
	ce, ok := ev.(Event)
	if !ok {
		return false
	}

	return e.Header.Equal(&ce.Header) && bytes.Equal(e.Payload.Bytes(), ce.Payload.Bytes())
}
