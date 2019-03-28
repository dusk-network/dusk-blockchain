package notary

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// LaunchSignatureSetNotary creates and ignites a SigSetNotary by injecting the EventBus, the Committee and the message validation primitive
func LaunchSignatureSetNotary(eventBus *wire.EventBus, c committee.Committee, currentRound uint64) *sigSetNotary {

	ssn := &sigSetNotary{
		eventBus:        eventBus,
		sigSetCollector: initSigSetCollector(eventBus, c, currentRound),
	}
	go ssn.Listen()
	ssn.sigSetCollector.RoundChan <- currentRound
	return ssn
}

type (
	// SigSetEvent is a CommitteeEvent decorated with the signature set hash
	SigSetEvent struct {
		*committee.NotaryEvent
		SigSetHash []byte
	}

	// sigSetNotary creates the proper EventSubscriber to listen to the SigSetEvent notification. It is not supposed to be used directly
	sigSetNotary struct {
		eventBus        *wire.EventBus
		sigSetCollector *sigSetCollector
	}

	SigSetEventUnmarshaller struct {
		*committee.NotaryEventUnMarshaller
	}

	// sigSetCollector collects SigSetEvent and decides whether it needs to propagate a round update. It gets the Quorum from the Committee interface and communicates through a channel to notify of round increments. Whenever it gets messages from future rounds, it queues them until their round becomes the current one.
	// The SigSetEvents for the current round are grouped by step. Future messages are grouped by round number
	// Finally a round update should be propagated when we get enough SigSetEvent messages for a given step
	sigSetCollector struct {
		*committee.Collector
		RoundChan    chan uint64
		futureRounds map[uint64][]*SigSetEvent
		Unmarshaller wire.EventUnmarshaller
	}
)

// NewSigSetEvent creates a SigSetEvent
func NewSigSetEvent() *SigSetEvent {
	return &SigSetEvent{NotaryEvent: committee.NewNotaryEvent()}
}

// Equal as specified in the Event interface
func (sse *SigSetEvent) Equal(e wire.Event) bool {
	return sse.NotaryEvent.Equal(e) && bytes.Equal(sse.SigSetHash, e.(*SigSetEvent).SigSetHash)
}

// Listen triggers the EventSubscriber to accept Events from the EventBus
func (ssn *sigSetNotary) Listen() {
	for {
		select {
		// When it is the right moment, the collector publishes round updates to the round channel
		case newRound := <-ssn.sigSetCollector.RoundChan:
			// Marshalling the round update
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, newRound)
			buf := bytes.NewBuffer(b)
			// publishing to the EventBus
			ssn.eventBus.Publish(msg.RoundUpdateTopic, buf)
		}
	}
}

func NewSigSetEventUnmarshaller() *SigSetEventUnmarshaller {
	return &SigSetEventUnmarshaller{
		NotaryEventUnMarshaller: committee.NewNotaryEventUnMarshaller(
			reduction.NewSigSetUnMarshaller(nil), msg.VerifyEd25519Signature),
	}
}

// Unmarshal as specified in the Event interface
func (sseu *SigSetEventUnmarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	// if the type checking is unsuccessful, it means that the injection is wrong. So panic!
	sigSetEv := ev.(*SigSetEvent)

	if err := sseu.EventUnMarshaller.Unmarshal(r, sigSetEv.NotaryEvent); err != nil {
		return err
	}

	if err := encoding.Read256(r, &sigSetEv.SigSetHash); err != nil {
		return err
	}

	return nil
}

// newSigSetCollector accepts a committee, a channel whereto publish the result and a validateFunc
func newSigSetCollector(c committee.Committee, currentRound uint64) *sigSetCollector {

	cc := &committee.Collector{
		StepEventCollector: make(map[string][]wire.Event),
		Committee:          c,
		CurrentRound:       currentRound,
	}
	return &sigSetCollector{
		Collector:    cc,
		RoundChan:    make(chan uint64),
		futureRounds: make(map[uint64][]*SigSetEvent),
		Unmarshaller: NewSigSetEventUnmarshaller(),
	}
}

// initSigSetCollector creates a SigSetCollector while also firing the proper subscriber the collector needs to listen to. In this case it listens to SigSetAgreementTopic
func initSigSetCollector(eventBus *wire.EventBus, c committee.Committee, currentRound uint64) *sigSetCollector {
	// creating the collector used in the EventSubscriber
	sigSetCollector := newSigSetCollector(c, currentRound)
	// creating the EventSubscriber listening to msg.SigSetAgreementTopic
	go wire.NewEventSubscriber(eventBus, sigSetCollector, string(msg.SigSetAgreementTopic)).Accept()
	return sigSetCollector
}

// Collect as specified in the EventCollector interface. It uses SigSetEvent.Unmarshal to populate the fields from the buffer and then it calls Process
func (s *sigSetCollector) Collect(buffer *bytes.Buffer) error {
	ev := NewSigSetEvent()
	if err := s.Unmarshaller.Unmarshal(buffer, ev); err != nil {
		return err
	}

	s.Process(ev)
	return nil
}

// ShouldBeStored checks if the message should be stored either in the current round queue or among future messages
func (s *sigSetCollector) ShouldBeStored(m *SigSetEvent) bool {
	step := m.Step
	sigSetList := s.Collector.StepEventCollector[string(step)]
	return len(sigSetList)+1 < s.Committee.Quorum() || m.Round > s.CurrentRound
}

// Process is a recursive function that checks whether the SigSetEvent notified should be ignored, stored or should trigger a round update. In the latter event, after notifying the round update in the proper channel and incrementing the round, it starts processing events which became relevant for this round
func (s *sigSetCollector) Process(ev *SigSetEvent) {
	isIrrelevant := s.CurrentRound != 0 && s.CurrentRound > ev.Round
	if s.ShouldBeSkipped(ev.NotaryEvent) || isIrrelevant {
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

		s.Store(ev, string(ev.Step))
		return
	}

	s.nextRound()
}

func (s *sigSetCollector) nextRound() {
	s.UpdateRound(s.CurrentRound + 1)
	// notify the Notary
	go func() { s.RoundChan <- s.CurrentRound }()
	s.Clear()

	//picking messages related to next round (now current)
	currentEvents := s.futureRounds[s.CurrentRound]
	//processing messages store so far
	for _, event := range currentEvents {
		s.Process(event)
	}
}
