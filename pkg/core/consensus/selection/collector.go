package selection

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type (
	// collector is the collector of SigSetEvents. It manages the EventQueue and the Selector to pick the best signature for a given step
	collector struct {
		CurrentRound     uint64
		CurrentStep      uint8
		CurrentBlockHash []byte
		BestEventChan    chan *bytes.Buffer
		// this is the dump of future messages, categorized by different steps and different rounds
		queue   *consensus.EventQueue
		timeOut time.Duration
		sync.RWMutex
		selector *EventSelector
		handler  scoreEventHandler
	}

	scoreEventHandler interface {
		consensus.EventHandler
		UpdateBidList(user.BidList)
	}

	state struct {
		round uint64
		step  uint8
	}

	selectionCollector struct {
		selectionChan chan state
	}

	// EventSelector is a helper to help choosing
	EventSelector struct {
		repropagateChan chan wire.Event
		bestEventChan   chan wire.Event
		prioritizer     wire.EventPrioritizer
		// this field is for testing purposes only
		sync.RWMutex
		bestEvent wire.Event
	}
)

func (sc *selectionCollector) Collect(r *bytes.Buffer) error {
	round := binary.LittleEndian.Uint64(r.Bytes()[:8])
	step := uint8(r.Bytes()[8])

	state := state{
		round: round,
		step:  step,
	}
	sc.selectionChan <- state
	return nil
}

// NewCollector creates a new Collector
func newCollector(bestEventChan chan *bytes.Buffer, handler scoreEventHandler, timerLength time.Duration) *collector {
	return &collector{
		BestEventChan: bestEventChan,
		selector:      NewEventSelector(handler),
		handler:       handler,
		queue:         consensus.NewEventQueue(),
		timeOut:       timerLength,
		RWMutex:       sync.RWMutex{},
	}
}

// initCollector instantiates a Collector and triggers the related EventSubscriber. It is a helper method for those needing access to the Collector
func initCollector(handler scoreEventHandler, timeout time.Duration, eventBus *wire.EventBus, topic string) *collector {
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

	s.selector.RLock()
	bestEvent := s.selector.bestEvent
	s.selector.RUnlock()
	event := s.handler.Priority(bestEvent, ev)
	if event == bestEvent {
		return errors.New("score of received event is lower than our current best event")
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
		log.WithFields(log.Fields{
			"process":         "selection",
			"message round":   h.Round,
			"message step":    h.Step,
			"collector round": s.CurrentRound,
			"collector step":  s.CurrentStep,
		}).Debugln("discarding score message")
		return
	}

	// Not yet
	if s.isEarly(h.Round, h.Step) {
		log.WithFields(log.Fields{
			"process":         "selection",
			"message round":   h.Round,
			"message step":    h.Step,
			"collector round": s.CurrentRound,
			"collector step":  s.CurrentStep,
		}).Debugln("queueing score message")
		s.queue.PutEvent(h.Round, h.Step, ev)
		return
	}

	// Just right :)
	s.selector.CompareEvent(ev)
}

func (s *collector) isObsolete(round uint64, step uint8) bool {
	return round < s.CurrentRound || step < s.CurrentStep
}

func (s *collector) isEarly(round uint64, step uint8) bool {
	return round > s.CurrentRound || step > s.CurrentStep || s.selector == nil
}

// UpdateRound stops the selection and cleans up the queue up to the round specified.
func (s *collector) UpdateRound(round uint64) {
	s.selector.Lock()
	s.stopSelection()
	s.selector.Unlock()

	s.CurrentRound = round
	s.CurrentStep = 1
	s.queue.ConsumeUntil(s.CurrentRound)
}

// StartSelection starts the selection of the best SigSetEvent. It also consumes stored events related to the current step
func (s *collector) StartSelection() {
	log.WithFields(log.Fields{
		"process": "selection",
		"round":   s.CurrentRound,
		"step":    s.CurrentStep,
	}).Debugln("starting selection")
	// stopping the selector after timeout
	time.AfterFunc(s.timeOut, func() {
		s.selector.ReturnBest()
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
	ev := <-s.selector.bestEventChan
	buf := new(bytes.Buffer)
	if err := s.handler.Marshal(buf, ev); err == nil {
		s.BestEventChan <- buf
		return
	}
	s.BestEventChan <- buf
}

func (s *collector) stopSelection() {
	s.selector.bestEvent = nil
}

//NewEventSelector creates the Selector
func NewEventSelector(p wire.EventPrioritizer) *EventSelector {
	return &EventSelector{
		repropagateChan: make(chan wire.Event, 100),
		bestEventChan:   make(chan wire.Event),
		prioritizer:     p,
	}
}

func (s *EventSelector) CompareEvent(ev wire.Event) {
	s.Lock()
	defer s.Unlock()
	s.bestEvent = s.prioritizer.Priority(s.bestEvent, ev)
	if s.bestEvent == ev {
		s.repropagateChan <- ev
	}
}

func (s *EventSelector) ReturnBest() {
	s.Lock()
	defer s.Unlock()
	s.bestEventChan <- s.bestEvent
	s.bestEvent = nil
}
