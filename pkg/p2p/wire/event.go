package wire

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

type (

	// The Event is an Entity that represents the Messages travelling on the EventBus.
	// It would normally present always the same fields.
	Event interface {
		Sender() []byte
		Equal(Event) bool
	}

	// EventUnmarshaller unmarshals an Event from a buffer. Following Golang's way of
	// defining interfaces, it exposes an Unmarshal method which allows for flexibility
	// and reusability across all the different components that need to read the buffer
	// coming from the EventBus into different structs
	EventUnmarshaller interface {
		Unmarshal(bytes.Buffer, Event) error
	}

	// EventMarshaller is the specular operation of an EventUnmarshaller. Following
	// Golang's way of defining interfaces, it exposes a Marshal method which allows
	// for flexibility and reusability across all the different components that need to
	// read the buffer coming from the EventBus into different structs
	EventMarshaller interface {
		Marshal(*bytes.Buffer, Event) error
	}

	// EventUnMarshaller is a convenient interface providing both Marshalling and
	// Unmarshalling capabilities
	EventUnMarshaller interface {
		EventMarshaller
		EventUnmarshaller
	}

	// EventPrioritizer is used by the EventSelector to prioritize events
	// (normally to return the best collected after a timespan). Return true if the first element has priority over the second, false otherwise
	EventPrioritizer interface {
		Priority(Event, Event) bool
	}

	// EventVerifier is the interface to verify an Event
	EventVerifier interface {
		Verify(Event) error
	}

	// EventCollector is the interface for collecting Events. Pretty much processors
	// involves some degree of Event collection (either until a Quorum is reached or
	// until a Timeout). This Interface is typically implemented by a struct that will
	// perform some Event unmarshalling.
	EventCollector interface {
		Collect(bytes.Buffer) error
	}

	// EventDeserializer is the interface for those struct that allows deserialization of an event from scratch
	EventDeserializer interface {
		Deserialize(*bytes.Buffer) (Event, error)
	}

	// Store abstract retrieval of Events
	Store interface {
		Insert(Event, string) int
		Clear()
		Contains(Event, string) bool
		Get(string) []Event
		All() []Event
	}
)

// AddTopic is a convenience function to add a specified topic at the start of
// a buffer. This topic is later decoded by the peer when gossiping messages,
// to be put on the message header.
func AddTopic(m *bytes.Buffer, topic topics.Topic) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	topicBytes := topics.TopicToByteArray(topic)
	if _, err := buffer.Write(topicBytes[:]); err != nil {
		return nil, err
	}

	if _, err := buffer.Write(m.Bytes()); err != nil {
		return nil, err
	}

	return buffer, nil
}
