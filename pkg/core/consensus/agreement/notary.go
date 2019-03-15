package agreement

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// The Event is an Entity that represents the Messages travelling on the EventBus. It would normally present always the same fields. Following Golang's way of defining interfaces, it exposes an Unmarshal method which allows for flexibility and reusability across all the different components that need to read the buffer coming from the EventBus into different structs
type Event interface {
	Unmarshal(*bytes.Buffer) error
	Equal(Event) bool
}

// EventCollector is the interface for collecting Events. Pretty much processors involves some degree of Event collection (either until a Quorum is reached or until a Timeout). This Interface is typically implemented by a struct that will perform some Event unmarshalling.
type EventCollector interface {
	Collect(*bytes.Buffer) error
}

// StepEventCollector is an helper for common operations on stored Event Arrays
type StepEventCollector map[uint8][]Event

// Clear up the Collector
func (sec StepEventCollector) Clear() {
	for key := range sec {
		delete(sec, key)
	}
}

// Contains checks if we already collected this event
func (sec StepEventCollector) Contains(event Event, step uint8) bool {
	for _, stored := range sec[step] {
		if event.Equal(stored) {
			return true
		}
	}

	return false
}

// Store the Event keeping track of the step it belongs to
func (sec StepEventCollector) Store(event Event, step uint8) int {
	eventList := sec[step]
	if eventList == nil {
		// TODO: should this have the Quorum as limit
		eventList = make([]Event, 100)
	}

	// storing the agreement vote for the proper step
	eventList = append(eventList, event)
	sec[step] = eventList
	return len(eventList)
}

func (sec StepEventCollector) GetCommitteeEvent(buffer *bytes.Buffer, event Event) error {
	return event.Unmarshal(buffer)
}

// EventSubscriber accepts events from the EventBus and takes care of reacting on quit Events. It delegates the business logic to the EventCollector which is supposed to handle the incoming events
type EventSubscriber struct {
	eventBus       *wire.EventBus
	eventCollector EventCollector
	msgChan        <-chan *bytes.Buffer
	msgChanID      uint32
	quitChan       <-chan *bytes.Buffer
	quitChanID     uint32
	topic          string
}

// NewEventSubscriber creates the EventSubscriber listening to a topic on the EventBus. The EventBus, EventCollector and Topic are injected
func NewEventSubscriber(eventBus *wire.EventBus, collector EventCollector, topic string) *EventSubscriber {

	quitChan := make(chan *bytes.Buffer, 1)
	msgChan := make(chan *bytes.Buffer, 100)

	msgChanID := eventBus.Subscribe(topic, msgChan)
	quitChanID := eventBus.Subscribe(string(msg.QuitTopic), msgChan)

	return &EventSubscriber{
		eventBus:       eventBus,
		msgChan:        msgChan,
		msgChanID:      msgChanID,
		quitChan:       quitChan,
		quitChanID:     quitChanID,
		topic:          topic,
		eventCollector: collector,
	}
}

// Accept incoming (mashalled) Events on the topic of interest and dispatch them to the EventCollector.Collect
func (n *EventSubscriber) Accept() {
	for {
		select {
		case <-n.quitChan:
			n.eventBus.Unsubscribe(n.topic, n.msgChanID)
			n.eventBus.Unsubscribe(string(msg.QuitTopic), n.quitChanID)
			return
		case eventMsg := <-n.msgChan:
			n.eventCollector.Collect(eventMsg)
		}
	}
}
