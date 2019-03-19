package reduction

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// EventUnmarshaller unmarshals an Event from a buffer. Following Golang's way of defining interfaces, it exposes an Unmarshal method which allows for flexibility and reusability across all the different components that need to read the buffer coming from the EventBus into different structs
type EventUnmarshaller interface {
	Unmarshal(*bytes.Buffer, Event) error
}

// The Event is an Entity that represents the Messages travelling on the EventBus. It would normally present always the same fields.
type Event interface {
	Equal(Event) bool
}

// EventCollector is the interface for collecting Events. Pretty much processors involves some degree of Event collection (either until a Quorum is reached or until a Timeout). This Interface is typically implemented by a struct that will perform some Event unmarshalling.
type EventCollector interface {
	Collect(*bytes.Buffer) error
}

// reductionEventCollector is an helper for common operations on stored Event Arrays
type reductionEventCollector map[string][]Event

// Clear up the Collector
func (rec reductionEventCollector) Clear() {
	for key := range rec {
		delete(rec, key)
	}
}

// Contains checks if we already collected this event
func (rec reductionEventCollector) Contains(event Event, hash string) bool {
	for _, stored := range rec[hash] {
		if event.Equal(stored) {
			return true
		}
	}

	return false
}

// Store the Event keeping track of the step it belongs to. It silently ignores duplicates (meaning it does not store an event in case it is already found at the step specified). It returns the number of events stored at specified step *after* the store operation
func (rec reductionEventCollector) Store(event Event, hash string) int {
	eventList := rec[hash]
	if rec.Contains(event, hash) {
		return len(eventList)
	}

	if eventList == nil {
		eventList = make([]Event, 0, 100)
	}

	// storing the agreement vote for the proper step
	eventList = append(eventList, event)
	rec[hash] = eventList
	return len(eventList)
}

type reductionEvent struct {
	VotedHash  []byte
	SignedHash []byte
	PubKeyBLS  []byte
	Round      uint64
	Step       uint8
}

// Equal as specified in the Event interface
func (r *reductionEvent) Equal(e Event) bool {
	other, ok := e.(*reductionEvent)
	return ok && (bytes.Equal(r.PubKeyBLS, other.PubKeyBLS)) && (r.Round == other.Round) && (r.Step == other.Step)
}

type reductionEventUnmarshaller struct {
	validate func(*bytes.Buffer) error
}

func newReductionEventUnmarshaller(validate func(*bytes.Buffer) error) *reductionEventUnmarshaller {
	return &reductionEventUnmarshaller{validate}
}

// Unmarshal unmarshals the buffer into a CommitteeEvent
func (a *reductionEventUnmarshaller) Unmarshal(r *bytes.Buffer, ev Event) error {
	if err := a.validate(r); err != nil {
		return err
	}

	// if the injection is unsuccessful, panic
	reductionEvent := ev.(*reductionEvent)

	if err := encoding.Read256(r, &reductionEvent.VotedHash); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &reductionEvent.SignedHash); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &reductionEvent.PubKeyBLS); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &reductionEvent.Round); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &reductionEvent.Step); err != nil {
		return err
	}

	return nil
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
	quitChanID := eventBus.Subscribe(string(msg.QuitTopic), quitChan)

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
