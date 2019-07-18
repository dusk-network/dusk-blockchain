package selection

import (
	"bytes"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
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
	handler   ScoreEventHandler
	lock      sync.RWMutex
	bestEvent wire.Event
	state     consensus.State
	timer     *consensus.Timer
	running   bool
}

// newEventSelector creates the Selector and returns it.
func newEventSelector(publisher wire.EventPublisher, handler ScoreEventHandler,
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
		s.propagateCertificate(ev)
		s.setBestEvent(ev)
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

	s.publisher.Stream(string(topics.Gossip), msg)
}

func (s *eventSelector) propagateCertificate(ev wire.Event) {
	sev := ev.(*ScoreEvent)
	buf := new(bytes.Buffer)
	if err := encoding.Write256(buf, sev.PrevHash); err != nil {
		panic(err)
	}

	if err := sev.Certificate.Encode(buf); err != nil {
		panic(err)
	}

	s.publisher.Publish(string(topics.Certificate), buf)
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
	s.publisher.Publish(msg.BestScoreTopic, buf)
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
