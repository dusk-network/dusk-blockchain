package notary

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// SigSetEvent is a CommitteeEvent decorated with the signature set hash
type SigSetEvent struct {
	*committeeEvent
	sigSetHash []byte
}

// Equal as specified in the Event interface
func (sse *SigSetEvent) Equal(e Event) bool {
	return sse.committeeEvent.Equal(e) && bytes.Equal(sse.sigSetHash, e.(*SigSetEvent).sigSetHash)
}

type sigSetEventUnmarshaller struct {
	*committeeEventUnmarshaller
}

func newSigSetEventUnmarshaller(validate func(*bytes.Buffer) error) *committeeEventUnmarshaller {
	return &committeeEventUnmarshaller{validate}
}

// Unmarshal as specified in the Event interface
func (sseu *sigSetEventUnmarshaller) Unmarshal(r *bytes.Buffer, ev Event) error {
	// if the type checking is unsuccessful, it means that the injection is wrong. So panic!
	sigSetEv := ev.(*SigSetEvent)

	if err := sseu.committeeEventUnmarshaller.Unmarshal(r, sigSetEv.committeeEvent); err != nil {
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
	*CommitteeCollector
	roundChan    chan<- uint64
	futureRounds map[uint64][]*SigSetEvent
	Unmarshaller EventUnmarshaller
}

// NewSigSetCollector accepts a committee, a channel whereto publish the result and a validateFunc
func NewSigSetCollector(committee user.Committee, roundChan chan uint64, validateFunc func(*bytes.Buffer) error, currentRound uint64) *SigSetCollector {

	cc := &CommitteeCollector{
		StepEventCollector: make(map[uint8][]Event),
		committee:          committee,
		currentRound:       currentRound,
	}
	return &SigSetCollector{
		CommitteeCollector: cc,
		roundChan:          roundChan,
		futureRounds:       make(map[uint64][]*SigSetEvent),
		Unmarshaller:       newSigSetEventUnmarshaller(validateFunc),
	}
}

// Collect as specified in the EventCollector interface. It uses SigSetEvent.Unmarshal to populate the fields from the buffer and then it calls Process
func (s *SigSetCollector) Collect(buffer *bytes.Buffer) error {
	ev := &SigSetEvent{committeeEvent: &committeeEvent{}}
	if err := s.Unmarshaller.Unmarshal(buffer, ev); err != nil {
		return err
	}

	s.Process(ev)
	return nil
}

// ShouldBeStored checks if the message should be stored either in the current round queue or among future messages
func (s *SigSetCollector) ShouldBeStored(m *SigSetEvent) bool {
	step := m.Step
	sigSetList := s.CommitteeCollector.StepEventCollector[step]
	return len(sigSetList)+1 < s.committee.Quorum() || m.Round > s.currentRound
}

// Process is a recursive function that checks whether the SigSetEvent notified should be ignored, stored or should trigger a round update. In the latter event, after notifying the round update in the proper channel and incrementing the round, it starts processing events which became relevant for this round
func (s *SigSetCollector) Process(event *SigSetEvent) {
	isIrrelevant := s.currentRound != 0 && s.currentRound < event.Round
	if s.ShouldBeSkipped(event.committeeEvent) || isIrrelevant {
		return
	}

	if s.ShouldBeStored(event) {
		if event.Round > s.currentRound {
			events := s.futureRounds[event.Round]
			if events == nil {
				events = make([]*SigSetEvent, 0, s.committee.Quorum())
			}
			events = append(events, event)
			s.futureRounds[event.Round] = events
			return
		}

		s.Store(event, event.Step)
		return
	}

	s.nextRound()
}

func (s *SigSetCollector) nextRound() {
	s.currentRound = s.currentRound + 1
	// notify the Notary
	go func() { s.roundChan <- s.currentRound }()
	s.Clear()

	currentEvents := s.futureRounds[s.currentRound]
	for _, event := range currentEvents {
		s.Process(event)
	}
}

// SigSetNotary creates the proper EventSubscriber to listen to the SigSetEvent notifications
type SigSetNotary struct {
	eventBus         *wire.EventBus
	sigSetSubscriber *EventSubscriber
	roundChan        <-chan uint64
	sigSetCollector  *SigSetCollector
}

// NewSigSetNotary creates a SigSetNotary by injecting the EventBus, the Committee and the message validation primitive
func NewSigSetNotary(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error, committee user.Committee, currentRound uint64) *SigSetNotary {

	//creating the channel whereto notifications about round updates are push onto
	roundChan := make(chan uint64, 1)
	// creating the collector used in the EventSubscriber
	sigSetCollector := NewSigSetCollector(committee, roundChan, validateFunc, currentRound)
	// creating the EventSubscriber listening to msg.SigSetAgreementTopic
	sigSetSubscriber := NewEventSubscriber(eventBus, sigSetCollector, string(msg.SigSetAgreementTopic))

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
