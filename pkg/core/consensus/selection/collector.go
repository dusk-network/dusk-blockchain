package selection

import (
	"bytes"
	"encoding/binary"
	"sync"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

var empty = new(struct{})

type (
	// collector is the collector of ScoreEvents. It manages the EventQueue and the
	// Selector to pick the best signature for a given step
	collector struct {
		state         consensus.State
		BestEventChan chan *bytes.Buffer

		// this is the dump of future messages, categorized by different steps and
		// different rounds
		queue *consensus.EventQueue

		selector *eventSelector
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

	eventSelector struct {
		publisher  wire.EventPublisher
		handler    scoreEventHandler
		marshaller wire.EventMarshaller
		sync.RWMutex
		bestEvent wire.Event
		state     consensus.State
		timer     *consensus.Timer
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
func newCollector(publisher wire.EventPublisher, handler scoreEventHandler,
	timerLength time.Duration, state consensus.State) *collector {
	return &collector{
		selector: newEventSelector(publisher, newScoreHandler(), timerLength, state),
		handler:  handler,
		queue:    consensus.NewEventQueue(),
		state:    state,
	}
}

// initCollector instantiates a Collector and triggers the related TopicListener. It is a helper method for those needing access to the Collector
func initCollector(handler scoreEventHandler, timeout time.Duration,
	eventBroker wire.EventBroker, topic string, state consensus.State) *collector {
	collector := newCollector(eventBroker, handler, timeout, state)
	go wire.NewTopicListener(eventBroker, collector, topic).Accept()
	return collector
}

// Collect as specified in the EventCollector interface. It delegates unmarshalling to the SigSetUnMarshaller which also validates the signatures.
func (s *collector) Collect(r *bytes.Buffer) error {
	ev := s.handler.NewEvent()

	if err := s.handler.Unmarshal(r, ev); err != nil {
		panic(err)
	}

	s.process(*ev.(*ScoreEvent))
	return nil
}

// process sends SigSetEvent to the selector. The messages have already been validated and verified
func (s *collector) process(ev ScoreEvent) {
	h := &consensus.EventHeader{}
	s.handler.ExtractHeader(ev, h)

	// Not anymore
	if s.isObsolete(h.Round) {
		log.WithFields(log.Fields{
			"process":         "selection",
			"message round":   h.Round,
			"message step":    h.Step,
			"collector state": s.state.String(),
		}).Debugln("discarding score message")
		return
	}

	// Not yet
	if s.isEarly(h.Round) {
		log.WithFields(log.Fields{
			"process":         "selection",
			"message round":   h.Round,
			"message step":    h.Step,
			"collector state": s.state.String(),
		}).Debugln("queueing score message")
		s.queue.PutEvent(h.Round, h.Step, ev)
		return
	}

	// Just right :)
	s.selector.compareEvent(ev)
}

func (s *collector) flushQueue() {
	queued, step := s.queue.ConsumeNextStepEvents(s.state.Round())
	if step == s.state.Step() {
		for _, ev := range queued {
			s.process(ev.(ScoreEvent))
		}
	}
}

func (s *collector) isObsolete(round uint64) bool {
	return round < s.state.Round()
}

func (s *collector) isEarly(round uint64) bool {
	return round > s.state.Round()
}

// UpdateRound stops the selection and cleans up the queue up to the round specified.
func (s *collector) UpdateRound(round uint64) {
	log.WithFields(log.Fields{
		"process": "selection",
		"round":   round,
	}).Debugln("updating round")
	s.selector.stopSelection()
	s.state.Update(round)
	s.queue.ConsumeUntil(s.state.Round())
	s.flushQueue()
	go s.selector.startSelection()
}

//NeweventSelector creates the Selector
func newEventSelector(publisher wire.EventPublisher, handler scoreEventHandler,
	timeOut time.Duration, state consensus.State) *eventSelector {
	return &eventSelector{
		publisher: publisher,
		handler:   handler,
		timer: &consensus.Timer{
			Timeout:     timeOut,
			TimeoutChan: make(chan interface{}, 1),
		},
		state:      state,
		marshaller: newScoreHandler(),
	}
}

// StartSelection starts the selection of the best ScoreEvent.
func (s *eventSelector) startSelection() {
	log.WithFields(log.Fields{
		"process":        "selection",
		"selector state": s.state.String(),
	}).Debugln("starting selection")
	// empty timeoutchan
	for len(s.timer.TimeoutChan) > 0 {
		<-s.timer.TimeoutChan
	}
	// propagating the best event after timeout,
	// or stopping on reading from timeoutchan
	timer := time.NewTimer(s.timer.Timeout)
	select {
	case <-timer.C:
		s.propagateBestEvent()
	case <-s.timer.TimeoutChan:
		s.Lock()
		defer s.Unlock()
		s.bestEvent = nil
	}
}

func (s *eventSelector) compareEvent(ev wire.Event) {
	s.RLock()
	bestEvent := s.bestEvent
	s.RUnlock()
	newBestEvent := s.handler.Priority(bestEvent, ev)
	if newBestEvent.Equal(ev) {
		if err := s.handler.Verify(ev); err != nil {
			log.WithField("process", "selection").Debugln(err)
			return
		}

		buf := new(bytes.Buffer)
		s.marshaller.Marshal(buf, newBestEvent)
		msg, _ := wire.AddTopic(buf, topics.Score)
		s.publisher.Publish(string(topics.Gossip), msg)
		s.Lock()
		defer s.Unlock()
		s.bestEvent = newBestEvent
	}
}

func (s *eventSelector) propagateBestEvent() {
	s.RLock()
	bestEvent := s.bestEvent
	s.RUnlock()
	buf := new(bytes.Buffer)
	s.marshaller.Marshal(buf, bestEvent)
	s.publisher.Publish(msg.BestScoreTopic, buf)
	s.state.IncrementStep()
}

func (s *eventSelector) stopSelection() {
	s.timer.TimeoutChan <- empty
}
