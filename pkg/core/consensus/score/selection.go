package score

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
	Prioritizer struct 

	// Collector is the collector of SigSetEvents. It manages the EventQueue and the Selector to pick the best signature for a given step
	Collector struct {
		CurrentRound  uint64
		CurrentStep   uint8
		BestEventChan chan *bytes.Buffer
		// this is the dump of future messages, categorized by different steps and different rounds
		queue        *consensus.EventQueue
		Unmarshaller wire.EventUnmarshaller
		Marshaller   wire.EventMarshaller

		selector    Selector
		timerLength time.Duration
	}

	// Broker is the component that supervises the collection of SigSetSelection events
	Broker struct {
		eventBus        *wire.EventBus
		phaseUpdateChan <-chan []byte
		roundUpdateChan <-chan uint64
		collector       *Collector
	}
)

func (p *Prioritizer) Priority(first wire.Event, second wire.Event) bool {
	ev1 := first.(Event)
	ev2 := second.(Event)
	return ev1.Score < ev2.Score
}

func NewSelector() {

}

// selectBestBlockHash will receive score messages from the inputChannel for a
// certain amount of time. It will store the highest score along with it's
// accompanied block hash, and will send the block hash associated to the highest
// score into the outputChannel once the timer runs out.
func (s *Selector) PickBest() {

}

// NewSigSetEvent is an alias for a committee.NewEvent. The alias would make it simpler to refactor in case we realize we need some customization in the SigSetEvent
var NewSigSetEvent = committee.NewEvent

// NewSigSetCollector creates a new Collector
func NewSigSetCollector(c committee.Committee, bestEventChan chan *bytes.Buffer, validateFunc func(*bytes.Buffer) error, timerLength time.Duration) *Collector {
	unMarshaller := committee.NewEventUnMarshaller(validateFunc)
	return &Collector{
		BestEventChan: bestEventChan,
		Unmarshaller:  unMarshaller,
		Marshaller:    unMarshaller,
	}
}

// InitSigSetCollector instantiates a Collector and triggers the related EventSubscriber. It is a helper method for those needing access to the Collector
func InitSigSetCollector(c committee.Committee, validateFunc func(*bytes.Buffer) error, timeout time.Duration, eventBus *wire.EventBus) *Collector {
	bestEventChan := make(chan *bytes.Buffer, 1)
	collector := NewSigSetCollector(c, bestEventChan, validateFunc, timeout)
	go wire.NewEventSubscriber(eventBus, collector, string(msg.SigSetSelectionTopic)).Accept()
	return collector
}

// Collect as specified in the EventCollector interface. It delegates unmarshalling to the SigSetUnMarshaller which also validates the signatures.
func (s *Collector) Collect(r *bytes.Buffer) error {
	ev := NewSigSetEvent()
	if err := s.Unmarshaller.Unmarshal(r, ev); err != nil {
		return err
	}

	// delegating the committee to verify the vote set
	//TODO: verify stuff


	s.process(ev)
	return nil
}

// process sends SigSetEvent to the selector. The messages have already been validated and verified
func (s *Collector) process(ev *Event) {
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

func (s *Collector) isObsolete(ev *Event) bool {
	return ev.Round < s.CurrentRound || ev.Step < s.CurrentStep
}

func (s *Collector) isEarly(ev *Event) bool {
	return ev.Round > s.CurrentRound || ev.Step > s.CurrentStep || s.selector == nil
}

// UpdateRound stops the selection and cleans up the queue up to the round specified.
func (s *Collector) UpdateRound(round uint64) {
	s.CurrentRound = round
	s.CurrentStep = 1
	s.stopSelection()
	s.queue.ConsumeUntil(s.CurrentRound)
}

// StartSelection starts the selection of the best SigSetEvent. It also consumes stored events related to the current step
func (s *Collector) StartSelection(blockHash []byte) {
	s.CurrentBlockHash = blockHash
	// creating a new selector
	//TODO: create selector
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

// listenSelection triggers a goroutine that notifies the Broker through its channel after having incremented the Collector step
func (s *Collector) listenSelection() {
	ev := <-s.selector.BestEventChan
	s.CurrentStep++
	sigSetEvent := ev.(*SigSetEvent)
	b := make([]byte, 32)
	buf := bytes.NewBuffer(b)
	s.Marshaller.Marshal(buf, sigSetEvent)
	s.BestEventChan <- buf
}

// stopSelection notifies the Selector to stop selecting
func (s *Collector) stopSelection() {
	if s.selector != nil {
		s.selector.StopChan <- true
		s.selector = nil
	}
}

// NewSigSetFilter creates a Broker component which responsibility is to listen to the eventbus and supervise Collector operations
func NewSigSetFilter(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error, c committee.Committee, timeout time.Duration) *Broker {
	//creating the channel whereto notifications about round updates are push onto

	roundChan := consensus.InitRoundCollector(eventBus).RoundChan
	phaseChan := consensus.InitPhaseCollector(eventBus).BlockHashChan
	collector := InitSigSetCollector(c, validateFunc, timeout, eventBus)

	return &Broker{
		eventBus:        eventBus,
		collector:       collector,
		roundUpdateChan: roundChan,
		phaseUpdateChan: phaseChan,
	}
}

// Listen on the eventBus for relevant topics to feed the collector
func (f *Broker) Listen() {
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
