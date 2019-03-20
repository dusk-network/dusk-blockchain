package selection

import (
	"bytes"
	"errors"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type (
	// SigSetEvent expresses a vote on a block hash. It is a real type alias of notaryEvent
	SigSetEvent = committee.Event

	// SigSetUnMarshaller is the unmarshaller of BlockEvents. It is a real type alias of notaryEventUnmarshaller
	SigSetUnMarshaller = committee.EventUnMarshaller

	// SigSetCollector is the collector of SigSetEvents. It manages the EventQueue and the Selector to pick the best signature for a given step
	SigSetCollector struct {
		Committee        committee.Committee
		CurrentBlockHash []byte
		CurrentRound     uint64
		CurrentStep      uint8
		BestEventChan    chan *bytes.Buffer
		// this is the dump of future messages, categorized by different steps and different rounds
		queue        *consensus.EventQueue
		Unmarshaller wire.EventUnmarshaller
		Marshaller   wire.EventMarshaller

		selector    *committee.Selector
		timerLength time.Duration
	}

	// SigSetFilter is the component that supervises the collection of SigSetSelection events
	SigSetFilter struct {
		eventBus        *wire.EventBus
		phaseUpdateChan <-chan []byte
		roundUpdateChan <-chan uint64
		collector       *SigSetCollector
	}
)

// NewSigSetEvent is an alias for a committee.NewEvent. The alias would make it simpler to refactor in case we realize we need some customization in the SigSetEvent
var NewSigSetEvent = committee.NewEvent

// NewSigSetCollector creates a new Collector
func NewSigSetCollector(c committee.Committee, bestEventChan chan *bytes.Buffer, validateFunc func(*bytes.Buffer) error, timerLength time.Duration) *SigSetCollector {
	unMarshaller := committee.NewEventUnMarshaller(validateFunc)
	return &SigSetCollector{
		BestEventChan: bestEventChan,
		Unmarshaller:  unMarshaller,
		Marshaller:    unMarshaller,
	}
}

// InitSigSetCollector instantiates a SigSetCollector and triggers the related EventSubscriber. It is a helper method for those needing access to the Collector
func InitSigSetCollector(c committee.Committee, validateFunc func(*bytes.Buffer) error, timeout time.Duration, eventBus *wire.EventBus) *SigSetCollector {
	bestEventChan := make(chan *bytes.Buffer, 1)
	collector := NewSigSetCollector(c, bestEventChan, validateFunc, timeout)
	go wire.NewEventSubscriber(eventBus, collector, string(msg.SigSetSelectionTopic)).Accept()
	return collector
}

// Collect as specified in the EventCollector interface. It delegates unmarshalling to the SigSetUnMarshaller which also validates the signatures.
func (s *SigSetCollector) Collect(r *bytes.Buffer) error {
	ev := NewSigSetEvent()
	if err := s.Unmarshaller.Unmarshal(r, ev); err != nil {
		return err
	}

	// delegating the committee to verify the vote set
	if err := s.Committee.VerifyVoteSet(ev.VoteSet, ev.SignedVoteSet, ev.Round, ev.Step); err != nil {
		return err
	}

	// validating if the event is related to the current winning block hash
	if !bytes.Equal(s.CurrentBlockHash, ev.BlockHash) {
		return errors.New("vote set is for the wrong block hash")
	}

	if len(ev.VoteSet) < s.Committee.Quorum() {
		// TODO: should we serialize the event into a string?
		return errors.New("signature selection: vote set is too small")
	}

	s.process(ev)
	return nil
}

// process sends SigSetEvent to the selector. The messages have already been validated and verified
func (s *SigSetCollector) process(ev *SigSetEvent) {
	// Not anymore
	if s.isObsolete(ev) {
		return
	}

	// Not yet
	if s.isEarly(ev) {
		s.queue.PutEvent(ev.Round, ev.Step, ev)
		return
	}

	// Just right :)
	s.selector.EventChan <- ev
}

func (s *SigSetCollector) isObsolete(ev *SigSetEvent) bool {
	return ev.Round < s.CurrentRound || ev.Step < s.CurrentStep
}

func (s *SigSetCollector) isEarly(ev *SigSetEvent) bool {
	return ev.Round > s.CurrentRound || ev.Step > s.CurrentStep || s.selector == nil
}

// UpdateRound stops the selection and cleans up the queue up to the round specified.
func (s *SigSetCollector) UpdateRound(round uint64) {
	s.CurrentRound = round
	s.CurrentStep = 1
	s.stopSelection()
	s.queue.ConsumeUntil(s.CurrentRound)
}

// StartSelection starts the selection of the best SigSetEvent. It also consumes stored events related to the current step
func (s *SigSetCollector) StartSelection(blockHash []byte) {
	s.CurrentBlockHash = blockHash
	// creating a new selector
	s.selector = committee.NewSelector(s.Committee, s.timerLength)
	// letting selector collect the best
	go s.selector.PickBest()
	// listening to the selector and collect its pick
	go s.listenSelection()

	// flushing the queue and processing relevant events
	queued, step := s.queue.ConsumeNextStepEvents(s.CurrentRound)
	if step == s.CurrentStep {
		for _, ev := range queued {
			s.process(ev.(*SigSetEvent))
		}
	}
}

// listenSelection triggers a goroutine that notifies the SigSetFilter through its channel after having incremented the Collector step
func (s *SigSetCollector) listenSelection() {
	ev := <-s.selector.BestEventChan
	s.CurrentStep++
	sigSetEvent := ev.(*SigSetEvent)
	b := make([]byte, 32)
	buf := bytes.NewBuffer(b)
	s.Marshaller.Marshal(buf, sigSetEvent)
	s.BestEventChan <- buf
}

// stopSelection notifies the Selector to stop selecting
func (s *SigSetCollector) stopSelection() {
	if s.selector != nil {
		s.selector.StopChan <- true
		s.selector = nil
	}
}

// NewSigSetFilter creates a SigSetFilter component which responsibility is to listen to the eventbus and supervise SigSetCollector operations
func NewSigSetFilter(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error, c committee.Committee, timeout time.Duration) *SigSetFilter {
	//creating the channel whereto notifications about round updates are push onto

	roundChan := consensus.InitRoundCollector(eventBus).RoundChan
	phaseChan := consensus.InitPhaseCollector(eventBus).BlockHashChan
	collector := InitSigSetCollector(c, validateFunc, timeout, eventBus)

	return &SigSetFilter{
		eventBus:        eventBus,
		collector:       collector,
		roundUpdateChan: roundChan,
		phaseUpdateChan: phaseChan,
	}
}

// Listen on the eventBus for relevant topics to feed the collector
func (f *SigSetFilter) Listen() {
	for {
		select {
		case roundUpdate := <-f.roundUpdateChan:
			f.collector.UpdateRound(roundUpdate)
		case phaseUpdate := <-f.phaseUpdateChan:
			f.collector.StartSelection(phaseUpdate)
		case bestEvent := <-f.collector.BestEventChan:
			f.eventBus.Publish(string(msg.SelectionResultTopic), bestEvent)
		}
	}
}
