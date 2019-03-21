package selection

import (
	"bytes"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

type (
	// EventHandler encapsulate logic specific to the various Collectors. Each Collector needs to verify, prioritize and extract information from Events. EventHandler is the interface that abstracts these operations away. The implementors of this interface is the real differentiator of the various consensus components
	EventHandler interface {
		wire.EventVerifier
		wire.EventPrioritizer
		NewEvent() wire.Event
		Stage(wire.Event) (uint64, uint8)
	}

	// Collector is the collector of SigSetEvents. It manages the EventQueue and the Selector to pick the best signature for a given step
	Collector struct {
		CurrentRound     uint64
		CurrentStep      uint8
		CurrentBlockHash []byte
		BestEventChan    chan *bytes.Buffer
		// this is the dump of future messages, categorized by different steps and different rounds
		queue        *consensus.EventQueue
		Unmarshaller wire.EventUnmarshaller
		Marshaller   wire.EventMarshaller

		selector    *wire.EventSelector
		handler     EventHandler
		timerLength time.Duration
	}

	// Broker is the component that supervises a collection of events
	Broker struct {
		eventBus        *wire.EventBus
		phaseUpdateChan <-chan []byte
		roundUpdateChan <-chan uint64
		collector       *Collector
		topic           string
	}
)

// NewCollector creates a new Collector
func NewCollector(bestEventChan chan *bytes.Buffer, validateFunc func(*bytes.Buffer) error, handler EventHandler, timerLength time.Duration) *Collector {
	unMarshaller := NewUnMarshaller(validateFunc)
	return &Collector{
		BestEventChan: bestEventChan,
		Unmarshaller:  unMarshaller,
		Marshaller:    unMarshaller,
		selector:      nil,
		handler:       handler,
	}
}

// InitCollector instantiates a Collector and triggers the related EventSubscriber. It is a helper method for those needing access to the Collector
func InitCollector(validateFunc func(*bytes.Buffer) error, handler EventHandler, timeout time.Duration, eventBus *wire.EventBus) *Collector {
	bestEventChan := make(chan *bytes.Buffer, 1)
	collector := NewCollector(bestEventChan, validateFunc, handler, timeout)
	go wire.NewEventSubscriber(eventBus, collector, string(topics.Score)).Accept()
	return collector
}

// Collect as specified in the EventCollector interface. It delegates unmarshalling to the SigSetUnMarshaller which also validates the signatures.
func (s *Collector) Collect(r *bytes.Buffer) error {
	ev := s.handler.NewEvent()

	if err := s.Unmarshaller.Unmarshal(r, ev); err != nil {
		return err
	}

	// verify
	if err := s.handler.Verify(ev); err != nil {
		return err
	}

	s.process(ev)
	return nil
}

// process sends SigSetEvent to the selector. The messages have already been validated and verified
func (s *Collector) process(ev wire.Event) {
	round, step := s.handler.Stage(ev)

	// Not anymore
	if s.isObsolete(round, step) {
		return
	}

	// Not yet
	if s.isEarly(round, step) {
		s.queue.PutEvent(round, step, ev)
		return
	}

	// Just right :)
	s.selector.EventChan <- ev
}

func (s *Collector) isObsolete(round uint64, step uint8) bool {
	return round < s.CurrentRound || step < s.CurrentStep
}

func (s *Collector) isEarly(round uint64, step uint8) bool {
	return round > s.CurrentRound || step > s.CurrentStep || s.selector == nil
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
	s.selector = wire.NewEventSelector(s.handler)
	// letting selector collect the best
	go s.selector.PickBest()
	// listening to the selector and collect its pick
	go s.listenSelection()

	// flushing the queue and processing relevant events
	queued, step := s.queue.ConsumeNextStepEvents(s.CurrentRound)
	if step == s.CurrentStep {
		for _, ev := range queued {
			s.process(ev)
		}
	}
}

// listenSelection triggers a goroutine that notifies the Broker through its channel after having incremented the Collector step
func (s *Collector) listenSelection() {
	ev := <-s.selector.BestEventChan
	s.CurrentStep++
	b := make([]byte, 32)
	buf := bytes.NewBuffer(b)
	s.Marshaller.Marshal(buf, ev)
	s.BestEventChan <- buf
}

// stopSelection notifies the Selector to stop selecting
func (s *Collector) stopSelection() {
	if s.selector != nil {
		s.selector.StopChan <- true
		s.selector = nil
	}
}

// NewBroker creates a Broker component which responsibility is to listen to the eventbus and supervise Collector operations
func NewBroker(eventBus *wire.EventBus, validateFunc func(*bytes.Buffer) error, handler EventHandler, timeout time.Duration, topic string) *Broker {
	//creating the channel whereto notifications about round updates are push onto
	roundChan := consensus.InitRoundCollector(eventBus).RoundChan
	phaseChan := consensus.InitPhaseCollector(eventBus).BlockHashChan
	collector := InitCollector(validateFunc, handler, timeout, eventBus)

	return &Broker{
		eventBus:        eventBus,
		collector:       collector,
		roundUpdateChan: roundChan,
		phaseUpdateChan: phaseChan,
		topic:           topic,
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
			f.eventBus.Publish(f.topic, bestEvent)
		}
	}
}
