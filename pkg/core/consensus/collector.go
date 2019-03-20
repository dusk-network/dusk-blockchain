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
		PubKeyBLS []byte
		Round     uint64
		Step      uint8
	}

	// EventHeaderMarshaller marshals a consensus EventHeader as follows:
	// - BLS Public Key
	// - Round
	// - Step
	EventHeaderMarshaller struct{}

	// EventHeaderUnmarshaller unmarshals consensus events. It is a helper to be embedded in the various consensus message unmarshallers
	EventHeaderUnmarshaller struct {
		validate func(*bytes.Buffer) error
	}

	// EventQueue is a Queue of Events grouped by rounds and steps
	EventQueue map[uint64]map[uint8][]wire.Event
)

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
func NewEventHeaderUnmarshaller(validate func(*bytes.Buffer) error) *EventHeaderUnmarshaller {
	return &EventHeaderUnmarshaller{validate}
}

// Unmarshal unmarshals the buffer into a ConsensusEvent
func (a *EventHeaderUnmarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	if err := a.validate(r); err != nil {
		return err
	}

	// if the injection is unsuccessful, panic
	consensusEv := ev.(*EventHeader)

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

// RoundCollector is a simple wrapper over a channel to get round notifications
type RoundCollector struct {
	RoundChan chan uint64
}

func InitRoundCollector(eventBus *wire.EventBus) *RoundCollector {
	roundChan := make(chan uint64, 1)
	roundCollector := &RoundCollector{roundChan}
	go wire.NewEventSubscriber(eventBus, roundCollector,
		string(msg.RoundUpdateTopic)).Accept()
	return roundCollector
}

// Collect as specified in the EventCollector interface. In this case Collect simply performs unmarshalling of the round event
func (r *RoundCollector) Collect(roundBuffer *bytes.Buffer) error {
	round := binary.LittleEndian.Uint64(roundBuffer.Bytes())
	r.RoundChan <- round
	return nil
}

type PhaseCollector struct {
	BlockHashChan chan []byte
}

func InitPhaseCollector(eventBus *wire.EventBus) *PhaseCollector {
	phaseUpdateChan := make(chan []byte)
	collector := &PhaseCollector{phaseUpdateChan}
	go wire.NewEventSubscriber(eventBus, collector,
		string(msg.SigSetAgreementTopic)).Accept()
	return collector
}

func (p *PhaseCollector) Collect(phaseBuffer *bytes.Buffer) error {
	p.BlockHashChan <- phaseBuffer.Bytes()
	return nil
}
