package selection

import (
	"bytes"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type selector struct {
	publisher eventbus.Publisher
	handler   *scoreHandler
	lock      sync.RWMutex
	bestEvent *ScoreEvent

	timer   *timer
	timeout time.Duration

	stepper consensus.Stepper
	signer  consensus.Signer
}

// NewComponent creates and launches the component which responsibility is to validate
// and select the best score among the blind bidders. The component publishes under
// the topic BestScoreTopic
func NewComponent(publisher eventbus.Publisher, timeout time.Duration) *selector {
	return &selector{
		timeout:   timeout,
		publisher: publisher,
	}
}

func (s *selector) Initialize(stepper consensus.Stepper, signer consensus.Signer, subscribe consensus.Subscriber, r consensus.RoundUpdate) []consensus.TopicListener {
	s.stepper = stepper
	s.signer = signer
	s.handler = newScoreHandler(r.BidList)

	scoreListener, _ := consensus.NewFilteringListener(s.CollectScoreEvent, s.Filter)
	scoreSubscriber := consensus.TopicListener{
		Topic:    topics.Score,
		Listener: scoreListener,
	}

	regenListener, _ := consensus.NewSimpleListener(s.CollectRegeneration)
	regenSubscriber := consensus.TopicListener{
		Topic:    topics.Regeneration,
		Listener: regenListener,
	}

	return []consensus.TopicListener{scoreSubscriber, regenSubscriber}
}

func (s *selector) Finalize() {
	s.stopSelection()
}

func (s *selector) CollectScoreEvent(e consensus.Event) error {
	ev := &ScoreEvent{}
	if err := UnmarshalScoreEvent(&e.Payload, ev); err != nil {
		return err
	}

	return s.Process(ev)
}

// Filter always returns false, as we don't do committee checks on bidders.
// Bidder relevant checks like proof verifications are done further down the pipeline
// (see `Process`).
// Implements consensus.Component
func (s *selector) Filter(hdr header.Header) bool {
	return false
}

func (s *selector) CollectRegeneration(e consensus.Event) error {
	s.handler.LowerThreshold()
	s.IncreaseTimeOut()
	s.setBestEvent(nil)
	s.startSelection()
	return nil
}

func (s *selector) startSelection() {
	s.timer.start(s.timeout)
}

func (s *selector) stopSelection() {
	s.timer.stop()
}

func (s *selector) Process(ev *ScoreEvent) error {
	if s.handler.Priority(s.getBestEvent(), ev) {
		// if the current best score has priority, we return
		return nil
	}

	if err := s.handler.Verify(ev); err != nil {
		return err
	}

	if err := s.repropagate(ev); err != nil {
		return err
	}

	s.setBestEvent(ev)
	return nil
}

func (s *selector) IncreaseTimeOut() {
	s.timeout = s.timeout * 2
}

func (s *selector) publishBestEvent() error {
	buf := new(bytes.Buffer)
	bestEvent := s.getBestEvent()

	s.signer.SendWithHeader(topics.BestScore, bestEvent.VoteHash, buf)
	s.stepper.RequestStepUpdate()
	return nil
}

func (s *selector) repropagate(ev *ScoreEvent) error {
	buf := topics.Score.ToBuffer()
	if err := MarshalScoreEvent(&buf, ev); err != nil {
		return err
	}

	s.publisher.Publish(topics.Gossip, &buf)
	return nil
}

func (s *selector) getBestEvent() *ScoreEvent {
	s.lock.RLock()
	defer s.lock.RUnlock()
	bestEvent := s.bestEvent
	return bestEvent
}

func (s *selector) setBestEvent(ev *ScoreEvent) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.bestEvent = ev
}
