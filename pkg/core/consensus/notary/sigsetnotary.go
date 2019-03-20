package notary

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// SigSetEvent is a CommitteeEvent decorated with the signature set hash
type SigSetEvent struct {
	*committee.Event
	sigSetHash []byte
}

// NewSigSetEvent creates a SigSetEvent
func NewSigSetEvent() *SigSetEvent {
	return &SigSetEvent{Event: committee.NewEvent()}
}

// Equal as specified in the Event interface
func (sse *SigSetEvent) Equal(e wire.Event) bool {
	return sse.Event.Equal(e) && bytes.Equal(sse.sigSetHash, e.(*SigSetEvent).sigSetHash)
}

type sigSetEventUnmarshaller struct {
	*committee.EventUnMarshaller
}

func newSigSetEventUnmarshaller(validate func(*bytes.Buffer) error) *sigSetEventUnmarshaller {
	return &sigSetEventUnmarshaller{
		EventUnMarshaller: committee.NewEventUnMarshaller(validate),
	}
}

// Unmarshal as specified in the Event interface
func (sseu *sigSetEventUnmarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	// if the type checking is unsuccessful, it means that the injection is wrong. So panic!
	sigSetEv := ev.(*SigSetEvent)

	if err := sseu.EventUnMarshaller.Unmarshal(r, sigSetEv.Event); err != nil {
		return err
	}

	if err := encoding.Read256(r, &sigSetEv.sigSetHash); err != nil {
		return err
	}

	return nil
}

// SigSetCollector collects SigSetEvent and decides whether it needs to propagate a round update. It gets the Quorum from the Committee interface and communicates through a channel to notify of round increments. Whenever it gets messages from future rounds, it queues them until their round becomes the current one.
// The SigSetEvents for the current round are grouped by step. Future messages are grouped by round number
// Finally a round update should be propagated when we get enough SigSetEvent messages for a given step
type SigSetCollector struct {
	*committee.Collector
	roundChan    chan<- uint64
	futureRounds map[uint64][]*SigSetEvent
	Unmarshaller wire.EventUnmarshaller
}

// NewSigSetCollector accepts a committee, a channel whereto publish the result and a validateFunc
func NewSigSetCollector(c committee.Committee, roundChan chan uint64, validateFunc func(*bytes.Buffer) error, currentRound uint64) *SigSetCollector {

	cc := &committee.Collector{
		StepEventCollector: make(map[uint8][]wire.Event),
		Committee:          c,
		CurrentRound:       currentRound,
	}
	return &SigSetCollector{
		Collector:    cc,
		roundChan:    roundChan,
		futureRounds: make(map[uint64][]*SigSetEvent),
		Unmarshaller: newSigSetEventUnmarshaller(validateFunc),
	}
}

// Collect as specified in the EventCollector interface. It uses SigSetEvent.Unmarshal to populate the fields from the buffer and then it calls Process
func (s *SigSetCollector) Collect(buffer *bytes.Buffer) error {
	ev := NewSigSetEvent()
	if err := s.Unmarshaller.Unmarshal(buffer, ev); err != nil {
		return err
	}

	s.Process(ev)
	return nil
}

// ShouldBeStored checks if the message should be stored either in the current round queue or among future messages
func (s *SigSetCollector) ShouldBeStored(m *SigSetEvent) bool {
	step := m.Step
	sigSetList := s.Collector.StepEventCollector[step]
	return len(sigSetList)+1 < s.Committee.Quorum() || m.Round > s.CurrentRound
}

// Process is a recursive function that checks whether the SigSetEvent notified should be ignored, stored or should trigger a round update. In the latter event, after notifying the round update in the proper channel and incrementing the round, it starts processing events which became relevant for this round
func (s *SigSetCollector) Process(ev *SigSetEvent) {
	isIrrelevant := s.CurrentRound != 0 && s.CurrentRound > ev.Round
	if s.ShouldBeSkipped(ev.Event) || isIrrelevant {
		return
	}

	if s.ShouldBeStored(ev) {
		if ev.Round > s.CurrentRound {
			//rounds in the future should be handled later. For now we just store messages related to future rounds
			events := s.futureRounds[ev.Round]
			if events == nil {
				events = make([]*SigSetEvent, 0, s.Committee.Quorum())
			}
			events = append(events, ev)
			s.futureRounds[ev.Round] = events
			return
		}

		s.Store(ev, ev.Step)
		return
	}

	s.nextRound()
}

func (s *SigSetCollector) nextRound() {
	s.UpdateRound(s.CurrentRound + 1)
	// notify the Notary
	go func() { s.roundChan <- s.CurrentRound }()
	s.Clear()

	//picking messages related to next round (now current)
	currentEvents := s.futureRounds[s.CurrentRound]
	//processing messages store so far
	for _, event := range currentEvents {
		s.Process(event)
	}
}

// SigSetNotary creates the proper EventSubscriber to listen to the SigSetEvent notifications
type SigSetNotary struct {
	eventBus         *wire.EventBus
	sigSetSubscriber *wire.EventSubscriber
	roundChan        <-chan uint64
	sigSetCollector  *SigSetCollector
}

// NewSigSetNotary creates a SigSetNotary by injecting the EventBus, the Committee and the message validation primitive
func NewSigSetNotary(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error, c committee.Committee, currentRound uint64) *SigSetNotary {

	//creating the channel whereto notifications about round updates are push onto
	roundChan := make(chan uint64, 1)
	// creating the collector used in the EventSubscriber
	sigSetCollector := NewSigSetCollector(c, roundChan, validateFunc, currentRound)
	// creating the EventSubscriber listening to msg.SigSetAgreementTopic
	sigSetSubscriber := wire.NewEventSubscriber(eventBus, sigSetCollector, string(msg.SigSetAgreementTopic))

	return &SigSetNotary{
		eventBus:         eventBus,
		sigSetSubscriber: sigSetSubscriber,
		roundChan:        roundChan,
		sigSetCollector:  sigSetCollector,
	}
}

// Listen triggers the EventSubscriber to accept Events from the EventBus
func (ssn *SigSetNotary) Listen() {
	// firing the goroutine to let the EventSubscriber listen to the topic of interest
	go ssn.sigSetSubscriber.Accept()
	for {
		select {
		// When it is the right moment, the collector publishes round updates to the round channel
		case newRound := <-ssn.roundChan:
			// Marshalling the round update
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, newRound)
			buf := bytes.NewBuffer(b)
			// publishing to the EventBus
			ssn.eventBus.Publish(msg.RoundUpdateTopic, buf)
		}
	}
}
