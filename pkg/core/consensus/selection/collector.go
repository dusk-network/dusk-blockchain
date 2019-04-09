package selection

import (
	"bytes"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type (
	// collector is the collector of SigSetEvents. It manages the EventQueue and the Selector to pick the best signature for a given step
	collector struct {
		CurrentRound     uint64
		CurrentStep      uint8
		CurrentBlockHash []byte
		BestEventChan    chan *bytes.Buffer
		RepropagateChan  chan wire.Event
		// this is the dump of future messages, categorized by different steps and different rounds
		queue    *consensus.EventQueue
		timeOut  time.Duration
		selector *EventSelector
		handler  consensus.EventHandler
	}

	selectionCollector struct {
		selectionChan chan bool
	}

	// EventSelector is a helper to help choosing
	EventSelector struct {
		EventChan       chan wire.Event
		RepropagateChan chan wire.Event
		BestEventChan   chan wire.Event
		StopChan        chan bool
		prioritizer     wire.EventPrioritizer
		// this field is for testing purposes only
		BestEvent wire.Event
	}
)

func (sc *selectionCollector) Collect(r *bytes.Buffer) error {
	sc.selectionChan <- true
	return nil
}

// NewCollector creates a new Collector
func newCollector(bestEventChan chan *bytes.Buffer, handler consensus.EventHandler, timerLength time.Duration) *collector {
	return &collector{
		BestEventChan:   bestEventChan,
		RepropagateChan: make(chan wire.Event, 100),
		selector:        nil,
		handler:         handler,
		queue:           &consensus.EventQueue{},
		timeOut:         timerLength,
	}
}

// initCollector instantiates a Collector and triggers the related EventSubscriber. It is a helper method for those needing access to the Collector
func initCollector(handler consensus.EventHandler, timeout time.Duration, eventBus *wire.EventBus, topic string) *collector {
	bestEventChan := make(chan *bytes.Buffer, 1)
	collector := newCollector(bestEventChan, handler, timeout)
	go wire.NewEventSubscriber(eventBus, collector, topic).Accept()
	return collector
}

// Collect as specified in the EventCollector interface. It delegates unmarshalling to the SigSetUnMarshaller which also validates the signatures.
func (s *collector) Collect(r *bytes.Buffer) error {
	ev := s.handler.NewEvent()

	if err := s.handler.Unmarshal(r, ev); err != nil {
		panic(err)
	}

	if s.selector != nil {
		event := s.handler.Priority(s.selector.BestEvent, ev)
		if event == s.selector.BestEvent {
			return errors.New("score of received event is lower than our current best event")
		}
	}

	// verify
	if err := s.handler.Verify(ev); err != nil {
		log.WithField("process", "selection").Debugln(err)
		return err
	}

	s.process(ev)
	return nil
}

// process sends SigSetEvent to the selector. The messages have already been validated and verified
func (s *collector) process(ev wire.Event) {
	h := &consensus.EventHeader{}
	s.handler.ExtractHeader(ev, h)

	// Not anymore
	if s.isObsolete(h.Round, h.Step) {
		return
	}

	// Not yet
	if s.isEarly(h.Round, h.Step) {
		s.queue.PutEvent(h.Round, h.Step, ev)
		return
	}

	// Just right :)
	s.selector.EventChan <- ev
}

func (s *collector) isObsolete(round uint64, step uint8) bool {
	return round < s.CurrentRound || step < s.CurrentStep
}

func (s *collector) isEarly(round uint64, step uint8) bool {
	return round > s.CurrentRound || step > s.CurrentStep || s.selector == nil
}

// UpdateRound stops the selection and cleans up the queue up to the round specified.
func (s *collector) UpdateRound(round uint64) {
	s.stopSelection()

	s.CurrentRound = round
	s.CurrentStep = 1
	s.queue.ConsumeUntil(s.CurrentRound)
}

// StartSelection starts the selection of the best SigSetEvent. It also consumes stored events related to the current step
func (s *collector) StartSelection() {
	log.WithField("process", "selection").Traceln("starting selection")
	// creating a new selector
	s.selector = NewEventSelector(s.handler, s.RepropagateChan)
	// letting selector collect the best
	go s.selector.PickBest()
	// stopping the selector after timeout
	time.AfterFunc(s.timeOut, func() {
		if s.selector != nil {
			s.selector.StopChan <- true
		}
	})
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
func (s *collector) listenSelection() {
	ev := <-s.selector.BestEventChan
	s.selector = nil
	buf := new(bytes.Buffer)
	if err := s.handler.Marshal(buf, ev); err == nil {
		s.BestEventChan <- buf
		return
	}
	s.BestEventChan <- buf
}

// stopSelection notifies the Selector to stop selecting
func (s *collector) stopSelection() {
	if s.selector != nil {
		s.selector.StopChan <- false
		s.selector = nil
	}
}

//NewEventSelector creates the Selector
func NewEventSelector(p wire.EventPrioritizer, repropagateChan chan wire.Event) *EventSelector {
	return &EventSelector{
		EventChan:       make(chan wire.Event, 100),
		RepropagateChan: repropagateChan,
		BestEventChan:   make(chan wire.Event, 1),
		StopChan:        make(chan bool, 1),
		prioritizer:     p,
		BestEvent:       nil,
	}
}

// PickBest picks the best event depending on the priority of the sender
func (s *EventSelector) PickBest() {
	for {
		select {
		case ev := <-s.EventChan:
			s.BestEvent = s.prioritizer.Priority(s.BestEvent, ev)
			// TODO: review - if this event replaced best event, repropagate
			if s.BestEvent == ev {
				s.RepropagateChan <- ev
			}
		case shouldNotify := <-s.StopChan:
			if shouldNotify {
				s.BestEventChan <- s.BestEvent
			}
			return
		}
	}
}
