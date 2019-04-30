package selection

import (
	"bytes"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func LaunchNotification(eventbus wire.EventSubscriber) <-chan *ScoreEvent {
	scoreChan := make(chan *ScoreEvent)
	evChan := consensus.LaunchNotification(eventbus,
		newScoreHandler(), msg.BestScoreTopic)

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
	publisher wire.EventPublisher
	handler   scoreEventHandler
	sync.RWMutex
	bestEvent wire.Event
	state     consensus.State
	timer     *consensus.Timer
	running   bool
}

//NeweventSelector creates the Selector
func newEventSelector(publisher wire.EventPublisher, handler scoreEventHandler,
	timeOut time.Duration, state consensus.State) *eventSelector {
	return &eventSelector{
		publisher: publisher,
		handler:   handler,
		timer: &consensus.Timer{
			Timeout:     timeOut,
			TimeoutChan: make(chan struct{}),
		},
		state: state,
	}
}

// StartSelection starts the selection of the best ScoreEvent.
func (s *eventSelector) startSelection() {
	s.Lock()
	s.running = true
	s.Unlock()
	log.WithFields(log.Fields{
		"process":        "selection",
		"selector state": s.state.String(),
	}).Debugln("starting selection")
	go func() {
		// propagating the best event after timeout,
		// or stopping on reading from timeoutchan
		timer := time.NewTimer(s.timer.Timeout)
		select {
		case <-timer.C:
			s.publishBestEvent()
		case <-s.timer.TimeoutChan:
			s.Lock()
			s.bestEvent = nil
			s.Unlock()
		}
		s.Lock()
		s.running = false
		s.Unlock()
	}()
}

func (s *eventSelector) Process(ev wire.Event) {
	s.RLock()
	bestEvent := s.bestEvent
	s.RUnlock()
	if !s.handler.Priority(bestEvent, ev) {
		if err := s.handler.Verify(ev); err != nil {
			log.WithField("process", "selection").Debugln(err)
			return
		}

		s.repropagate(ev)
		s.Lock()
		defer s.Unlock()
		s.bestEvent = ev
	}
}

func (s *eventSelector) repropagate(ev wire.Event) {
	buf := new(bytes.Buffer)
	if err := s.handler.Marshal(buf, ev); err != nil {
		panic(err)
	}

	msg, err := wire.AddTopic(buf, topics.Score)
	if err != nil {
		panic(err)
	}

	s.publisher.Publish(string(topics.Gossip), msg)
}

func (s *eventSelector) publishBestEvent() {
	s.RLock()
	bestEvent := s.bestEvent
	s.RUnlock()
	buf := new(bytes.Buffer)
	err := s.handler.Marshal(buf, bestEvent)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"process": "score selection",
		}).Warnln("Error in marshalling score")
		return
	}
	s.publisher.Publish(msg.BestScoreTopic, buf)
	s.state.IncrementStep()
	s.Lock()
	defer s.Unlock()
	s.bestEvent = nil
}

func (s *eventSelector) stopSelection() {
	select {
	case s.timer.TimeoutChan <- empty:
	default:
	}
}
