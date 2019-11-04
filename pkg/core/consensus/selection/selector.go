package selection

import (
	"bytes"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "selector")
var emptyScore = Score{}

// Selector is the component responsible to collect score events and propagate
// the best one after a timeout
type Selector struct {
	publisher eventbus.Publisher
	handler   Handler
	lock      sync.RWMutex
	bestEvent *Score

	timer   *timer
	timeout time.Duration

	scoreID uint32

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

// Initialize the Selector according to the `consensus.Component` interface
func (s *Selector) Initialize(eventPlayer consensus.EventPlayer, signer consensus.Signer, r consensus.RoundUpdate) []consensus.TopicListener {
	s.eventPlayer = eventPlayer
	s.signer = signer
	s.handler = NewScoreHandler(r.BidList)
	s.timer = &timer{s: s}

	scoreSubscriber := consensus.TopicListener{
		Topic:         topics.Score,
		Preprocessors: []eventbus.Preprocessor{&consensus.Validator{}},
		Listener:      consensus.NewSimpleListener(s.CollectScoreEvent, consensus.LowPriority, false),
	}
	s.scoreID = scoreSubscriber.ID()

	generationSubscriber := consensus.TopicListener{
		Topic:    topics.Generation,
		Listener: consensus.NewSimpleListener(s.CollectGeneration, consensus.HighPriority, false),
	}

	return []consensus.TopicListener{scoreSubscriber, generationSubscriber}
}

// ID of the selector is the ID of the score event listener
func (s *Selector) ID() uint32 {
	return s.scoreID
}

// Finalize pauses the `EventPlayer` and stops the timer
func (s *Selector) Finalize() {
	s.eventPlayer.Pause(s.scoreID)
	s.timer.stop()
}

// CollectScoreEvent checks the score of an incoming Event and, in case
// it has a higher score, verifies, propagates and saves it
func (s *Selector) CollectScoreEvent(e consensus.Event) error {
	ev := Score{}
	if err := UnmarshalScore(&e.Payload, &ev); err != nil {
		return err
	}

	// Locking here prevents the named pipe from filling up with requests to verify
	// Score messages with a low score, as only one Score message will be verified
	// at a time. Consequently, any lower scores are discarded.
	s.lock.Lock()
	defer s.lock.Unlock()
	// Only check for priority if we already have a best event
	if !s.bestEvent.Equal(emptyScore) {
		if s.handler.Priority(*s.bestEvent, ev) {
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

	lg.WithFields(log.Fields{
		"current best": s.bestEvent.Score,
		"new best":     ev.Score,
	}).Debugln("swapping best score")
	s.bestEvent = &ev
	return nil
}

// CollectGeneration signals the selection start by triggering
// `EventPlayer.Play`
func (s *Selector) CollectGeneration(e consensus.Event) error {
	s.setBestEvent(emptyScore)
	_ = s.eventPlayer.Play(s.ID())
	s.startSelection()
	return nil
}

func (s *Selector) startSelection() {
	// Empty queue in a goroutine to avoid letting other listeners wait
	go s.eventPlayer.Resume(s.scoreID)
	s.timer.start(s.timeout)
}

// IncreaseTimeOut increases the timeout after a failed selection
func (s *Selector) IncreaseTimeOut() {
	s.timeout = s.timeout * 2
}

func (s *Selector) publishBestEvent() error {
	s.eventPlayer.Pause(s.scoreID)
	buf := new(bytes.Buffer)
	bestEvent := s.getBestEvent()
	// If we had no best event, we should send an empty hash
	if bestEvent.Equal(emptyScore) {
		s.signer.SendWithHeader(topics.BestScore, make([]byte, 32), buf, s.ID())
	} else {
		s.signer.SendWithHeader(topics.BestScore, bestEvent.VoteHash, buf, s.ID())
	}

	s.handler.LowerThreshold()
	s.IncreaseTimeOut()
	return nil
}

func (s *Selector) repropagate(ev Score) error {
	buf := new(bytes.Buffer)
	if err := MarshalScore(buf, &ev); err != nil {
		return err
	}

	s.signer.SendAuthenticated(topics.Score, ev.VoteHash, buf, s.ID())
	return nil
}

func (s *Selector) getBestEvent() Score {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return *s.bestEvent
}

func (s *Selector) setBestEvent(ev Score) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.bestEvent = &ev
}
