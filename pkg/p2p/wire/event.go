package wire

import (
	"bytes"
)

// QuitTopic is the topic to make all components quit
const QuitTopic = "topic"

type (

	// The Event is an Entity that represents the Messages travelling on the EventBus. It would normally present always the same fields.
	Event interface {
		Sender() []byte
		Equal(Event) bool
	}

	// EventUnmarshaller unmarshals an Event from a buffer. Following Golang's way of defining interfaces, it exposes an Unmarshal method which allows for flexibility and reusability across all the different components that need to read the buffer coming from the EventBus into different structs
	EventUnmarshaller interface {
		Unmarshal(*bytes.Buffer, Event) error
	}

	// EventMarshaller is the specular operation of an EventUnmarshaller. Following Golang's way of defining interfaces, it exposes an Unmarshal method which allows for flexibility and reusability across all the different components that need to read the buffer coming from the EventBus into different structs
	EventMarshaller interface {
		Marshal(*bytes.Buffer, Event) error
	}

	EventPrioritizer interface {
		Priority(Event, Event) bool
	}

	// EventCollector is the interface for collecting Events. Pretty much processors involves some degree of Event collection (either until a Quorum is reached or until a Timeout). This Interface is typically implemented by a struct that will perform some Event unmarshalling.
	EventCollector interface {
		Collect(*bytes.Buffer) error
	}

	// StepEventCollector is an helper for common operations on stored Event Arrays
	// TODO: move this inside the collector
	StepEventCollector map[uint8][]Event

	// EventSubscriber accepts events from the EventBus and takes care of reacting on quit Events. It delegates the business logic to the EventCollector which is supposed to handle the incoming events
	EventSubscriber struct {
		eventBus       *EventBus
		eventCollector EventCollector
		msgChan        <-chan *bytes.Buffer
		msgChanID      uint32
		quitChan       <-chan *bytes.Buffer
		quitChanID     uint32
		topic          string
	}

	// EventSelector is a helper to help choosing
	EventSelector struct {
		EventChan     chan Event
		BestEventChan chan Event
		StopChan      chan bool
		prioritizer   EventPrioritizer
	}
)

//NewEventSelector creates the Selector
func NewEventSelector(p EventPrioritizer) *EventSelector {
	return &EventSelector{
		EventChan:     make(chan Event),
		BestEventChan: make(chan Event),
		StopChan:      make(chan bool),
		prioritizer:   p,
	}
}

// PickBest picks the best event depending on the priority of the sender
func (s *EventSelector) PickBest() {
	var bestEvent Event

	for {
		select {
		case ev := <-s.EventChan:
			if s.prioritizer.Priority(bestEvent, ev) {
				bestEvent = ev
			}
		case shouldNotify := <-s.StopChan:
			if shouldNotify {
				s.BestEventChan <- bestEvent
			}
			return
		}
	}
}

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

// Store the Event keeping track of the step it belongs to. It silently ignores duplicates (meaning it does not store an event in case it is already found at the step specified). It returns the number of events stored at specified step *after* the store operation
func (sec StepEventCollector) Store(event Event, step uint8) int {
	eventList := sec[step]
	if sec.Contains(event, step) {
		return len(eventList)
	}

	if eventList == nil {
		eventList = make([]Event, 0, 100)
	}

	// storing the agreement vote for the proper step
	eventList = append(eventList, event)
	sec[step] = eventList
	return len(eventList)
}

// NewEventSubscriber creates the EventSubscriber listening to a topic on the EventBus. The EventBus, EventCollector and Topic are injected
func NewEventSubscriber(eventBus *EventBus, collector EventCollector,
	topic string) *EventSubscriber {

	quitChan := make(chan *bytes.Buffer, 1)
	msgChan := make(chan *bytes.Buffer, 100)

	msgChanID := eventBus.Subscribe(topic, msgChan)
	quitChanID := eventBus.Subscribe(string(QuitTopic), quitChan)

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
			n.eventBus.Unsubscribe(string(QuitTopic), n.quitChanID)
			return
		case eventMsg := <-n.msgChan:
			n.eventCollector.Collect(eventMsg)
		}
	}
}
