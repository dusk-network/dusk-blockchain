package selection

import (
	"bytes"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

func LaunchNotification(eventbus eventbus.Subscriber) <-chan *ScoreEvent {
	scoreChan := make(chan *ScoreEvent)
	evChan := consensus.LaunchNotification(eventbus,
		newScoreHandler(), topics.BestScore)

	go func() {
		for {
			sEv := <-evChan
			scoreChan <- sEv.(*ScoreEvent)
		}
	}()

	return scoreChan
}

var empty struct{}

type eventSelector struct {
	publisher eventbus.Publisher
	handler   ScoreEventHandler
	lock      sync.RWMutex
	bestEvent wire.Event
	state     consensus.State
	timer     *consensus.Timer
	running   bool
}

// newEventSelector creates the Selector and returns it.
func newEventSelector(publisher eventbus.Publisher, handler ScoreEventHandler,
	timeOut time.Duration, state consensus.State) *eventSelector {
	return &eventSelector{
		publisher: publisher,
		handler:   handler,
		timer:     consensus.NewTimer(timeOut, make(chan struct{})),
		state:     state,
	}
}

// StartSelection starts the selection of the best ScoreEvent.
func (s *eventSelector) startSelection() {
	s.lock.Lock()
	s.running = true
	s.lock.Unlock()
	log.WithFields(log.Fields{
		"process":        "selection",
		"selector state": s.state.String(),
	}).Debugln("starting selection")
	go func() {
		// propagating the best event after timeout,
		// or stopping on reading from timeoutchan
		timer := time.NewTimer(s.timer.TimeOut())
		select {
		case <-timer.C:
			s.publishBestEvent()
		case <-s.timer.TimeOutChan:
		}
		s.lock.Lock()
		s.running = false
		s.lock.Unlock()
	}()
}

func (s *eventSelector) Process(ev wire.Event) {
	s.lock.RLock()
	bestEvent := s.bestEvent
	s.lock.RUnlock()
	if !s.handler.Priority(bestEvent, ev) {
		if err := s.handler.Verify(ev); err != nil {
			log.WithField("process", "selection").Debugln(err)
			return
		}

		s.repropagate(ev)
		s.setBestEvent(ev)
	}
}

func (s *eventSelector) repropagate(ev wire.Event) {
	buf := new(bytes.Buffer)
	if err := s.handler.Marshal(buf, ev); err != nil {
		panic(err)
	}
	if err := topics.Prepend(buf, topics.Score); err != nil {
		panic(err)
	}

	s.publisher.Publish(topics.Gossip, buf)
}

func (s *eventSelector) publishBestEvent() {
	s.lock.RLock()
	bestEvent := s.bestEvent
	s.lock.RUnlock()
	buf := new(bytes.Buffer)
	err := s.handler.Marshal(buf, bestEvent)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"process": "score selection",
		}).Warnln("Error in marshalling score")
		return
	}
	s.publisher.Publish(topics.BestScore, buf)
	s.state.IncrementStep()
	s.setBestEvent(nil)
}

func (s *eventSelector) stopSelection() {
	s.setBestEvent(nil)
	select {
	case s.timer.TimeOutChan <- empty:
	default:
	}
}

func (s *eventSelector) setBestEvent(ev wire.Event) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.bestEvent = ev
}

func (s *eventSelector) isRunning() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.running
}
