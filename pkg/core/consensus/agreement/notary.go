package agreement

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type Event interface {
	Unmarshal(*bytes.Buffer) error
	Equal(Event) bool
}

type EventCollector interface {
	Collect(*bytes.Buffer) error
}

type CommitteeEventCollector interface {
	Contains(*CommitteeEvent) bool
}

// StepEventCollector is an helper for common operations on stored events
type StepEventCollector map[uint8][]Event

func (sec StepEventCollector) Clear() {
	for key := range sec {
		delete(sec, key)
	}
}

// IsDuplicate checks if we already collected this event
func (sec StepEventCollector) Contains(event Event, step uint8) bool {
	for _, stored := range sec[step] {
		if event.Equal(stored) {
			return true
		}
	}

	return false
}

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

type EventSubscriber struct {
	eventBus       *wire.EventBus
	eventCollector EventCollector
	msgChan        <-chan *bytes.Buffer
	msgChanID      uint32
	quitChan       <-chan *bytes.Buffer
	quitChanID     uint32
	topic          string
}

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
