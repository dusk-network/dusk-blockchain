package notary

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
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

// A committeeEvent is an embeddable struct that groups common operations on notaries
type committeeEvent struct {
	VoteSet       []*msg.Vote
	SignedVoteSet []byte
	PubKeyBLS     []byte
	Round         uint64
	Step          uint8
	BlockHash     []byte
}

// Equal as specified in the Event interface
func (a *committeeEvent) Equal(e Event) bool {
	other, ok := e.(*committeeEvent)
	return ok && (bytes.Equal(a.PubKeyBLS, other.PubKeyBLS)) && (a.Round == other.Round) && (a.Step == other.Step)
}

type committeeEventUnmarshaller struct {
	validate func(*bytes.Buffer) error
}

func newCommitteeEventUnmarshaller(validate func(*bytes.Buffer) error) *committeeEventUnmarshaller {
	return &committeeEventUnmarshaller{validate}
}

// Unmarshal unmarshals the buffer into a CommitteeEvent
func (a *committeeEventUnmarshaller) Unmarshal(r *bytes.Buffer, ev Event) error {
	if err := a.validate(r); err != nil {
		return err
	}

	// if the injection is unsuccessful, panic
	committeeEvent := ev.(*committeeEvent)

	voteSet, err := msg.DecodeVoteSet(r)
	if err != nil {
		return err
	}
	committeeEvent.VoteSet = voteSet

	if err := encoding.ReadBLS(r, &committeeEvent.SignedVoteSet); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &committeeEvent.PubKeyBLS); err != nil {
		return err
	}

	if err := encoding.Read256(r, &committeeEvent.BlockHash); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &committeeEvent.Round); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &committeeEvent.Step); err != nil {
		return err
	}

	return nil
}

// CommitteeCollector is a helper that groups common operations performed on Events related to a committee
type CommitteeCollector struct {
	StepEventCollector
	committee    user.Committee
	currentRound uint64
}

// ShouldBeSkipped checks if the message is not propagated by a committee member, that is not a duplicate (and in this case should probably check if the Provisioner is malicious) and that is relevant to the current round
// NOTE: currentRound is handled by some other process, so it is not this component's responsibility to handle corner cases (for example being on an obsolete round because of a disconnect, etc)
func (cc *CommitteeCollector) ShouldBeSkipped(m *committeeEvent) bool {
	isDupe := cc.Contains(m, m.Step)
	isPleb := !cc.committee.IsMember(m.PubKeyBLS)
	//TODO: the round element needs to be reassessed
	err := cc.committee.VerifyVoteSet(m.VoteSet, m.BlockHash, m.Round, m.Step)
	failedVerification := err != nil
	return isDupe || isPleb || failedVerification
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
