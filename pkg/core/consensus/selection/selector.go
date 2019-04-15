package selection

import (
	"bytes"
	"sync"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

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
			TimeoutChan: make(chan struct{}, 1),
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
	// empty timeoutchan
	for len(s.timer.TimeoutChan) > 0 {
		<-s.timer.TimeoutChan
	}
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
}

func (s *eventSelector) Process(ev wire.Event) {
	s.RLock()
	bestEvent := s.bestEvent
	s.RUnlock()
	newBestEvent := s.handler.Priority(bestEvent, ev)
	if newBestEvent.Equal(ev) {
		if err := s.handler.Verify(ev); err != nil {
			log.WithField("process", "selection").Debugln(err)
			return
		}

		s.Lock()
		defer s.Unlock()
		s.bestEvent = newBestEvent
	}
}

func (s *eventSelector) publishBestEvent() {
	s.RLock()
	bestEvent := s.bestEvent
	s.RUnlock()
	buf := new(bytes.Buffer)
	s.handler.Marshal(buf, bestEvent)
	s.publisher.Publish(msg.BestScoreTopic, buf)
	s.state.IncrementStep()
	s.Lock()
	defer s.Unlock()
	s.bestEvent = nil
}

func (s *eventSelector) stopSelection() {
	s.timer.TimeoutChan <- empty
}
