package wire

import (
	"bytes"
	"io"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// QuitTopic is the topic to make all components quit
const QuitTopic = "quit"

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
		Unmarshal(*bytes.Buffer, Event) error
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
		Collect(*bytes.Buffer) error
	}

	TopicProcessor interface {
		Process(*bytes.Buffer) (*bytes.Buffer, error)
	}

	// TopicListener accepts events from the EventBus and takes care of reacting on
	// quit Events. It delegates the business logic to the EventCollector which is
	// supposed to handle the incoming events
	TopicListener struct {
		subscriber     EventSubscriber
		eventCollector EventCollector
		msgChan        <-chan *bytes.Buffer
		msgChanID      uint32
		quitChan       chan *bytes.Buffer
		quitChanID     uint32
		topic          string
	}

	// EventPreprocessors allow registration of preprocessors to be applied to incoming Event on a specific topic
	EventPreprocessor interface {
		RegisterPreprocessor(string, ...TopicProcessor) []uint32
		RemovePreprocessor(string, uint32)
		RemoveAllPreprocessors(string)
	}

	// EventSubscriber subscribes a channel to Event notifications on a specific topic
	EventSubscriber interface {
		EventPreprocessor
		Subscribe(string, chan<- *bytes.Buffer) uint32
		SubscribeCallback(string, func(*bytes.Buffer) error) uint32
		SubscribeStream(string, io.WriteCloser) uint32
		Unsubscribe(string, uint32)
		// RegisterPreprocessor(string, ...TopicProcessor)
	}

	// EventPublisher publishes serialized messages on a specific topic
	EventPublisher interface {
		Publish(string, *bytes.Buffer)
		Stream(string, *bytes.Buffer)
	}

	// EventBroker is an EventPublisher and an EventSubscriber
	EventBroker interface {
		EventSubscriber
		EventPublisher
	}

	// EventDeserializer is the interface for those struct that allows deserialization of an event from scratch
	EventDeserializer interface {
		Deserialize(*bytes.Buffer) (Event, error)
	}

	Store interface {
		Insert(Event, string) int
		Clear()
		Contains(Event, string) bool
		Get(string) []Event
		All() []Event
	}
)

// NewTopicListener creates the TopicListener listening to a topic on the EventBus.
// The EventBus, EventCollector and Topic are injected
func NewTopicListener(subscriber EventSubscriber, collector EventCollector, topic string,
	preprocessors ...TopicProcessor) *TopicListener {

	msgChan := make(chan *bytes.Buffer, 100)
	quitChan := make(chan *bytes.Buffer, 1)
	msgChanID := subscriber.Subscribe(topic, msgChan)
	quitChanID := subscriber.Subscribe(string(QuitTopic), quitChan)

	if len(preprocessors) > 0 {
		subscriber.RegisterPreprocessor(topic, preprocessors...)
	}

	return &TopicListener{
		subscriber:     subscriber,
		msgChan:        msgChan,
		msgChanID:      msgChanID,
		quitChan:       quitChan,
		quitChanID:     quitChanID,
		topic:          topic,
		eventCollector: collector,
	}
}

// Accept incoming (mashalled) Events on the topic of interest and dispatch them to the EventCollector.Collect.
func (n *TopicListener) Accept() {
	log.WithFields(log.Fields{
		"id":    n.msgChanID,
		"topic": n.topic,
	}).Debugln("Accepting messages")
	for {
		select {
		case <-n.quitChan:
			n.subscriber.Unsubscribe(n.topic, n.msgChanID)
			n.subscriber.Unsubscribe(string(QuitTopic), n.quitChanID)
			return
		case eventBuffer := <-n.msgChan:
			if len(n.msgChan) > 10 {
				log.WithFields(log.Fields{
					"id":         n.msgChanID,
					"topic":      n.topic,
					"Unconsumed": len(n.msgChan),
				}).Debugln("Channel is accumulating messages")
			}
			if err := n.eventCollector.Collect(eventBuffer); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"id":    n.msgChanID,
					"topic": n.topic,
				}).Errorln("Error in eventCollector.Collect")
			}
		}
	}
}

// Quit will kill the goroutine spawned by Accept, and unsubscribe from it's subscribed topics.
func (n *TopicListener) Quit() {
	n.quitChan <- new(bytes.Buffer)
}

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
