package consensus

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	// EventHeader is an embeddable struct representing the consensus event header fields
	EventHeader struct {
		// TODO: ed25519 related fields added for demo to facilitate easy
		// re-propagation. review
		Signature []byte
		PubKeyEd  []byte
		PubKeyBLS []byte
		Round     uint64
		Step      uint8
	}

	// EventHandler encapsulate logic specific to the various Collectors. Each Collector needs to verify, prioritize and extract information from Events. EventHandler is the interface that abstracts these operations away. The implementors of this interface is the real differentiator of the various consensus components
	EventHandler interface {
		wire.EventVerifier
		wire.EventPrioritizer
		wire.EventUnMarshaller
		NewEvent() wire.Event
		ExtractHeader(wire.Event, *EventHeader)
	}

	// EventHeaderMarshaller marshals a consensus EventHeader as follows:
	// - BLS Public Key
	// - Round
	// - Step
	EventHeaderMarshaller struct{}

	// EventHeaderUnmarshaller unmarshals consensus events. It is a helper to be embedded in the various consensus message unmarshallers
	EventHeaderUnmarshaller struct {
		Validate func([]byte, []byte, []byte) error
	}

	// EventQueue is a Queue of Events grouped by rounds and steps
	EventQueue map[uint64]map[uint8][]wire.Event

	// StepEventCollector is an helper for common operations on stored Event Arrays
	StepEventCollector map[string][]wire.Event

	// phaseCollector is not supposed to be used directly. Components interested in Phase Updates should import InitPhaseUpdate instead
	phaseCollector struct {
		blockHashChan chan []byte
	}

	// roundCollector is a simple wrapper over a channel to get round notifications. It is not supposed to be used directly. Components interestesd in Round updates should use InitRoundUpdate instead
	roundCollector struct {
		roundChan chan uint64
	}
)

// InitRoundUpdate initializes a Round update channel and fires up the EventSubscriber as well. Its purpose is to lighten up a bit the amount of arguments in creating the handler for the collectors. Also it removes the need to store subscribers on the consensus process
func InitRoundUpdate(eventBus *wire.EventBus) chan uint64 {
	roundChan := make(chan uint64, 1)
	roundCollector := &roundCollector{roundChan}
	go wire.NewEventSubscriber(eventBus, roundCollector,
		string(msg.RoundUpdateTopic)).Accept()
	return roundChan
}

// Equal as specified in the Event interface
func (a *EventHeader) Equal(e wire.Event) bool {
	other, ok := e.(*EventHeader)
	return ok && (bytes.Equal(a.PubKeyBLS, other.PubKeyBLS)) && (a.Round == other.Round) && (a.Step == other.Step)
}

// Sender of the Event
func (a *EventHeader) Sender() []byte {
	return a.PubKeyBLS
}

// Marshal an EventHeader into a Buffer
func (ehm *EventHeaderMarshaller) Marshal(r *bytes.Buffer, ev wire.Event) error {
	consensusEv := ev.(*EventHeader)

	// TODO: review. added to be able to marshal events without signature and
	// edwards pubkey, for signing purposes in the voting package.
	if consensusEv.Signature != nil && consensusEv.PubKeyEd != nil {
		if err := encoding.Write512(r, consensusEv.Signature); err != nil {
			return err
		}

		if err := encoding.Write256(r, consensusEv.PubKeyEd); err != nil {
			return err
		}
	}

	if err := encoding.WriteVarBytes(r, consensusEv.PubKeyBLS); err != nil {
		return err
	}

	if err := encoding.WriteUint64(r, binary.LittleEndian, consensusEv.Round); err != nil {
		return err
	}

	if err := encoding.WriteUint8(r, consensusEv.Step); err != nil {
		return err
	}

	return nil
}

// NewEventHeaderUnmarshaller creates an EventHeaderUnmarshaller delegating validation to the validate function
func NewEventHeaderUnmarshaller(validate func([]byte, []byte, []byte) error) *EventHeaderUnmarshaller {
	return &EventHeaderUnmarshaller{validate}
}

// Unmarshal unmarshals the buffer into a ConsensusEvent
func (a *EventHeaderUnmarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	// if the injection is unsuccessful, panic
	consensusEv := ev.(*EventHeader)

	// TODO: ed25519 related fields added for demo to facilitate easy
	// re-propagation. review
	if err := encoding.Read512(r, &consensusEv.Signature); err != nil {
		return err
	}

	if err := encoding.Read256(r, &consensusEv.PubKeyEd); err != nil {
		return err
	}

	// verify the signature here
	if err := a.Validate(consensusEv.PubKeyEd, r.Bytes(), consensusEv.Signature); err != nil {
		return err
	}

	// Decoding PubKey BLS
	if err := encoding.ReadVarBytes(r, &consensusEv.PubKeyBLS); err != nil {
		return err
	}

	// Decoding Round
	if err := encoding.ReadUint64(r, binary.LittleEndian, &consensusEv.Round); err != nil {
		return err
	}

	// Decoding Step
	if err := encoding.ReadUint8(r, &consensusEv.Step); err != nil {
		return err
	}

	return nil
}

// NewEventQueue creates a new EventQueue. It is primarily used by Collectors to temporarily store messages not yet relevant to the collection process
func NewEventQueue() EventQueue {
	ret := make(map[uint64]map[uint8][]wire.Event)
	return ret
}

// GetEvents returns the events for a round and step
func (s EventQueue) GetEvents(round uint64, step uint8) []wire.Event {
	if s[round][step] != nil {
		messages := s[round][step]
		s[round][step] = nil
		return messages
	}

	return nil
}

// PutEvent stores an Event at a given round and step
func (s EventQueue) PutEvent(round uint64, step uint8, m wire.Event) {
	// Initialise the map on this round if it was not yet created
	if s[round] == nil {
		s[round] = make(map[uint8][]wire.Event)
	}

	s[round][step] = append(s[round][step], m)
}

// Clear the queue
func (s EventQueue) Clear(round uint64) {
	s[round] = nil
}

// ConsumeNextStepEvents retrieves the Events stored at the lowest step for a passed round and returns them. The step gets deleted
func (s EventQueue) ConsumeNextStepEvents(round uint64) ([]wire.Event, uint8) {
	steps := s[round]
	if steps == nil {
		return nil, 0
	}

	nextStep := uint8(0)
	for k := range steps {
		if k > nextStep {
			nextStep = k
		}
	}

	events := s[round][nextStep]
	delete(s[round], nextStep)
	return events, nextStep
}

// ConsumeUntil consumes Events until the round specified (excluded). It returns the map slice deleted
func (s EventQueue) ConsumeUntil(round uint64) map[uint64]map[uint8][]wire.Event {
	ret := make(map[uint64]map[uint8][]wire.Event)
	for k := range s {
		if k < round {
			ret[k] = s[k]
		}
		delete(s, k)
	}
	return ret
}

// Clear up the Collector
func (sec StepEventCollector) Clear() {
	for key := range sec {
		delete(sec, key)
	}
}

// Contains checks if we already collected this event
func (sec StepEventCollector) Contains(event wire.Event, step string) bool {
	for _, stored := range sec[step] {
		if event.Equal(stored) {
			return true
		}
	}

	return false
}

// Store the Event keeping track of the step it belongs to. It silently ignores duplicates (meaning it does not store an event in case it is already found at the step specified). It returns the number of events stored at specified step *after* the store operation
func (sec StepEventCollector) Store(event wire.Event, step string) int {
	eventList := sec[step]
	if sec.Contains(event, step) {
		return len(eventList)
	}

	if eventList == nil {
		eventList = make([]wire.Event, 0, 100)
	}

	// storing the agreement vote for the proper step
	eventList = append(eventList, event)
	sec[step] = eventList
	return len(eventList)
}

// Collect as specified in the EventCollector interface. In this case Collect simply performs unmarshalling of the round event
func (r *roundCollector) Collect(roundBuffer *bytes.Buffer) error {
	round := binary.LittleEndian.Uint64(roundBuffer.Bytes())
	r.roundChan <- round
	return nil
}

func (p *phaseCollector) Collect(phaseBuffer *bytes.Buffer) error {
	p.blockHashChan <- phaseBuffer.Bytes()
	return nil
}
