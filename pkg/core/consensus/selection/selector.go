package selection

import (
	"bytes"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

type selector struct {
	publisher eventbus.Publisher
	handler   ScoreEventHandler
	lock      sync.RWMutex
	bestEvent wire.Event

	timer *timer

	requestStepUpdate func()
}

// Launch creates and launches the component which responsibility is to validate
// and select the best score among the blind bidders. The component publishes under
// the topic BestScoreTopic
func NewComponent(publisher eventbus.Publisher, keys user.Keys) {
	return newSelector(eventBroker)
}

func newSelector(publisher eventbus.Publisher) *selector {
	return &selector{
		publisher: publisher,
	}
}

func (s *selector) Initialize(r consensus.RoundUpdate) []consensus.Subscriber {
	s.handler = newScoreHandler(r.BidList)

	scoreSubscriber := consensus.Subscriber{
		topic:    topics.Score,
		listener: consensus.NewFilteringListener(s.CollectScoreEvent, r.Filter),
	}

	regenSubscriber := consensus.Subscriber{
		topic:    topics.Regeneration,
		listener: consensus.NewSimpleListener(s.CollectRegeneration),
	}

	return []consensus.Subscriber{scoreSubscriber, regenSubscriber}
}

func (s *selector) Finalize() {
	s.stopSelection()
}

func (s *selector) CollectScoreEvent(m bytes.Buffer, hdr header.Header) error {
	ev := ScoreEvent{}
	if err := UnmarshalScoreEvent(&m, &ev); err != nil {
		return err
	}

	s.Process(ev)
}

// Filter always returns false, as we don't do committee checks on bidders.
// Bidder relevant checks like proof verifications are done further down the pipeline
// (see `Process`).
// Implements consensus.Component
func (s *selector) Filter(hdr header.Header) bool {
	return false
}

// SetStep clears out the `bestEvent` field on the selector, as this event becomes
// irrelevant on a step update.
// Implements consensus.Component
func (s *selector) SetStep(step uint8) {
	s.setBestEvent(nil)
}

func (s *selector) CollectRegeneration(m bytes.Buffer, hdr header.Header) error {
	s.handler.LowerThreshold()
	s.timer.IncreaseTimeOut()
	s.startSelection()
	return nil
}

func (s *selector) startSelection() {
	s.timer.Start()
}

func (s *selector) stopSelection() {
	s.timer.Stop()
}

func (s *selector) Process(ev wire.Event) {
	if !s.handler.Priority(s.getBestEvent(), ev) {
		if err := s.handler.Verify(ev); err != nil {
			log.WithField("process", "selection").Debugln(err)
			return
		}

		s.repropagate(ev)
		s.setBestEvent(ev)
	}
}

func (s *selector) publishBestEvent() {
	buf := new(bytes.Buffer)
	if err := s.handler.Marshal(buf, s.getBestEvent()); err != nil {
		panic(err)
	}

	s.requestStepUpdate()
	s.publisher.Publish(topics.BestScore, buf)
}

func (s *selector) repropagate(ev wire.Event) {
	buf := topics.Score.ToBuffer()
	if err := s.handler.Marshal(&buf, ev); err != nil {
		panic(err)
	}

	s.publisher.Publish(topics.Gossip, &buf)
}

func (s *selector) getBestEvent() wire.Event {
	s.lock.RLock()
	defer s.lock.RUnlock()
	bestEvent := s.bestEvent
	return bestEvent
}

func (s *selector) setBestEvent(ev wire.Event) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.bestEvent = ev
}
