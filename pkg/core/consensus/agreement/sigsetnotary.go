package agreement

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type SigSetEvent struct {
	*CommitteeEvent
	sigSetHash []byte
}

func newSigSetEvent(validate func(*bytes.Buffer) error) *SigSetEvent {
	return &SigSetEvent{
		CommitteeEvent: newCommiteeEvent(validate),
	}
}

func (sse *SigSetEvent) Equal(e Event) bool {
	return sse.CommitteeEvent.Equal(e) && bytes.Equal(sse.sigSetHash, e.(*SigSetEvent).sigSetHash)
}

func (sse *SigSetEvent) Unmarshal(r *bytes.Buffer) error {
	if err := sse.CommitteeEvent.Unmarshal(r); err != nil {
		return err
	}

	if err := encoding.Read256(r, &sse.sigSetHash); err != nil {
		return err
	}

	return nil
}

type SigSetCollector struct {
	*CommitteeCollector
	roundChan    chan<- uint64
	futureRounds map[uint64][]*SigSetEvent
}

func NewSigSetCollector(committee user.Committee, roundChan chan uint64, validateFunc func(*bytes.Buffer) error) *SigSetCollector {

	cc := &CommitteeCollector{
		committee:    committee,
		validateFunc: validateFunc,
	}
	return &SigSetCollector{
		CommitteeCollector: cc,
		roundChan:          roundChan,
		futureRounds:       make(map[uint64][]*SigSetEvent),
	}
}

func (s *SigSetCollector) Collect(buffer *bytes.Buffer) error {
	ev := newSigSetEvent(s.validateFunc)
	if err := s.GetCommitteeEvent(buffer, ev); err != nil {
		return err
	}

	s.Process(ev)
	return nil
}

func (s *SigSetCollector) ShouldBeStored(m *SigSetEvent) bool {
	step := m.Step
	sigSetList := s.CommitteeCollector.StepEventCollector[step]
	return sigSetList == nil || len(sigSetList)+1 < s.committee.Quorum() || m.Round > s.currentRound
}

func (s *SigSetCollector) Process(event *SigSetEvent) {
	if s.ShouldBeSkipped(event.CommitteeEvent) {
		return
	}

	if s.ShouldBeStored(event) {
		if event.Round > s.currentRound {
			events := s.futureRounds[event.Round]
			if events == nil {
				events = make([]*SigSetEvent, s.committee.Quorum())
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

// SigSetNotary
type SigSetNotary struct {
	eventBus         *wire.EventBus
	sigSetSubscriber *EventSubscriber
	roundChan        <-chan uint64
}

// NewSigSetNotary creates a BlockNotary by injecting the EventBus, the CommitteeStore and the message validation primitive
func NewSigSetNotary(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error, committee user.Committee) *SigSetNotary {

	roundChan := make(chan uint64, 1)
	roundCollector := NewSigSetCollector(committee, roundChan, validateFunc)
	sigSetSubscriber := NewEventSubscriber(eventBus, roundCollector, string(msg.SigSetAgreementTopic))

	return &SigSetNotary{
		eventBus:         eventBus,
		sigSetSubscriber: sigSetSubscriber,
		roundChan:        roundChan,
	}
}

// Listen to block agreement messages and signature set agreement messages and propagates round and phase updates.
// A round update should be propagated when we get enough sigSetAgreement messages for a given step
// A phase update should be propagated when we get enough blockAgreement messages for a certain blockhash
// BlockNotary gets a currentRound somehow
func (ssn *SigSetNotary) Listen() {
	go ssn.sigSetSubscriber.Accept()
	for {
		select {
		case newRound := <-ssn.roundChan:
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, newRound)
			buf := bytes.NewBuffer(b)
			ssn.eventBus.Publish(msg.RoundUpdateTopic, buf)
		}
	}
}
