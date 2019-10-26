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

type Selector struct {
	publisher eventbus.Publisher
	handler   Handler
	lock      sync.RWMutex
	bestEvent *ScoreEvent

	timer   *timer
	timeout time.Duration

	eventPlayer consensus.EventPlayer
	signer      consensus.Signer
}

// NewComponent creates and launches the component which responsibility is to validate
// and select the best score among the blind bidders. The component publishes under
// the topic BestScoreTopic
func NewComponent(publisher eventbus.Publisher, timeout time.Duration) *Selector {
	return &Selector{
		timeout:   timeout,
		publisher: publisher,
	}
}

func (s *Selector) Initialize(eventPlayer consensus.EventPlayer, signer consensus.Signer, r consensus.RoundUpdate) []consensus.TopicListener {
	s.eventPlayer = eventPlayer
	s.signer = signer
	s.handler = NewScoreHandler(r.BidList)
	s.timer = &timer{s: s}

	scoreSubscriber := consensus.TopicListener{
		Topic:         topics.Score,
		Preprocessors: []eventbus.Preprocessor{&consensus.Validator{}},
		Listener:      consensus.NewFilteringListener(s.CollectScoreEvent, s.Filter),
	}

	regenSubscriber := consensus.TopicListener{
		Topic:    topics.Regeneration,
		Listener: consensus.NewSimpleListener(s.CollectRegeneration),
	}

	// We should trigger a selection on round update
	s.startSelection()
	return []consensus.TopicListener{scoreSubscriber, regenSubscriber}
}

func (s *Selector) Finalize() {
	s.stopSelection()
}

func (s *Selector) CollectScoreEvent(e consensus.Event) error {
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
func (s *Selector) Filter(hdr header.Header) bool {
	return false
}

func (s *Selector) CollectRegeneration(e consensus.Event) error {
	s.handler.LowerThreshold()
	s.IncreaseTimeOut()
	s.setBestEvent(nil)
	s.startSelection()
	return nil
}

func (s *Selector) startSelection() {
	s.timer.start(s.timeout)
}

func (s *Selector) stopSelection() {
	s.timer.stop()
}

func (s *Selector) Process(ev *ScoreEvent) error {
	bestEvent := s.getBestEvent()

	// Only check for priority if we already have a best event
	if bestEvent != nil {
		if s.handler.Priority(s.getBestEvent(), ev) {
			// if the current best score has priority, we return
			return nil
		}
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

func (s *Selector) IncreaseTimeOut() {
	s.timeout = s.timeout * 2
}

func (s *Selector) publishBestEvent() error {
	buf := new(bytes.Buffer)
	bestEvent := s.getBestEvent()
	// If we had no best event, we should send an empty hash
	if bestEvent == nil {
		s.signer.SendWithHeader(topics.BestScore, make([]byte, 32), buf)
	} else {
		s.signer.SendWithHeader(topics.BestScore, bestEvent.VoteHash, buf)
	}

	s.eventPlayer.Forward()
	return nil
}

func (s *Selector) repropagate(ev *ScoreEvent) error {
	buf := topics.Score.ToBuffer()
	if err := MarshalScoreEvent(&buf, ev); err != nil {
		return err
	}

	s.publisher.Publish(topics.Gossip, &buf)
	return nil
}

func (s *Selector) getBestEvent() *ScoreEvent {
	s.lock.RLock()
	defer s.lock.RUnlock()
	bestEvent := s.bestEvent
	return bestEvent
}

func (s *Selector) setBestEvent(ev *ScoreEvent) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.bestEvent = ev
}
