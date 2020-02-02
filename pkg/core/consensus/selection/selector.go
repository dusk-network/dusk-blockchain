package selection

import (
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "selector")
var emptyScore [32]byte

// Selector is the component responsible for collecting score events and propagating
// the best one after a timeout.
type Selector struct {
	publisher eventbus.Publisher
	handler   Handler
	lock      sync.RWMutex
	bestEvent message.Score

	timer   *timer
	timeout time.Duration

	scoreID uint32

	eventPlayer consensus.EventPlayer
	signer      consensus.Signer
}

type emptyScoreFactory struct {
}

func (e emptyScoreFactory) Create(pubkey []byte, round uint64, step uint8) consensus.InternalPacket {
	hdr := header.Header{
		Round:     round,
		Step:      step,
		PubKeyBLS: pubkey,
		BlockHash: emptyScore[:],
	}
	return message.EmptyScoreProposal(hdr)
}

// NewComponent creates and launches the component which responsibility is to validate
// and select the best score among the blind bidders. The component publishes under
// the topic BestScoreTopic
func NewComponent(publisher eventbus.Publisher, timeout time.Duration) *Selector {
	return &Selector{
		timeout:   timeout,
		publisher: publisher,
		bestEvent: message.EmptyScore(),
	}
}

// Initialize the Selector, by creating the handler and returning the needed Listeners.
// Implements consensus.Component.
func (s *Selector) Initialize(eventPlayer consensus.EventPlayer, signer consensus.Signer, r consensus.RoundUpdate) []consensus.TopicListener {
	s.eventPlayer = eventPlayer
	s.signer = signer
	s.handler = NewScoreHandler(r.BidList)
	s.timer = &timer{s: s}

	scoreSubscriber := consensus.TopicListener{
		Topic:    topics.Score,
		Listener: consensus.NewSimpleListener(s.CollectScoreEvent, consensus.LowPriority, false),
	}
	s.scoreID = scoreSubscriber.ID()

	generationSubscriber := consensus.TopicListener{
		Topic:    topics.Generation,
		Listener: consensus.NewSimpleListener(s.CollectGeneration, consensus.HighPriority, false),
	}

	return []consensus.TopicListener{scoreSubscriber, generationSubscriber}
}

// ID returns the ID of the Score message Listener.
// Implements consensus.Component
func (s *Selector) ID() uint32 {
	return s.scoreID
}

// Finalize pauses event streaming and stops the timer.
// Implements consensus.Component.
func (s *Selector) Finalize() {
	s.eventPlayer.Pause(s.scoreID)
	s.timer.stop()
}

// CollectScoreEvent checks the score of an incoming Event and, in case
// it has a higher score, verifies, propagates and saves it
func (s *Selector) CollectScoreEvent(packet consensus.InternalPacket) error {
	score := packet.(message.Score)

	// Locking here prevents the named pipe from filling up with requests to verify
	// Score messages with a low score, as only one Score message will be verified
	// at a time. Consequently, any lower scores are discarded.
	s.lock.Lock()
	defer s.lock.Unlock()
	// Only check for priority if we already have a best event
	if !s.bestEvent.IsEmpty() {
		if s.handler.Priority(s.bestEvent, score) {
			// if the current best score has priority, we return
			return nil
		}
	}

	if err := s.handler.Verify(score); err != nil {
		return err
	}

	// Tell the candidate broker to allow a candidate block with this
	// hash through.
	msg := message.New(topics.Score, score)
	s.publisher.Publish(topics.ValidCandidateHash, msg)

	if err := s.signer.Gossip(msg, s.ID()); err != nil {
		return err
	}

	lg.WithFields(log.Fields{
		"new best": score.Score,
	}).Debugln("swapping best score")
	s.bestEvent = score
	return nil
}

// CollectGeneration signals the selection start by triggering `EventPlayer.Play`
func (s *Selector) CollectGeneration(packet consensus.InternalPacket) error {
	s.lock.Lock()
	s.bestEvent = message.EmptyScore()
	s.lock.Unlock()
	_ = s.eventPlayer.Forward(s.ID())
	s.startSelection()
	return nil
}

func (s *Selector) startSelection() {
	// Empty queue in a goroutine to avoid letting other listeners wait
	go s.eventPlayer.Play(s.scoreID)
	s.timer.start(s.timeout)
}

// IncreaseTimeOut increases the timeout after a failed selection
func (s *Selector) IncreaseTimeOut() {
	s.timeout = s.timeout * 2
}

func (s *Selector) sendBestEvent() error {
	var bestEvent message.Score
	s.eventPlayer.Pause(s.scoreID)
	s.lock.RLock()
	bestEvent = s.bestEvent
	s.lock.RUnlock()

	// If we had no best event, we should send an empty hash
	if bestEvent.IsEmpty() {
		bestEvent = s.signer.Compose(emptyScoreFactory{}).(message.Score)
	}

	msg := message.New(topics.Score, bestEvent)
	s.signer.SendInternally(topics.BestScore, msg, s.ID())
	s.handler.LowerThreshold()
	s.IncreaseTimeOut()
	return nil
}

// repropagate tells the Coordinator/Signer to gossip the score
func (s *Selector) repropagate(ev message.Score) error {
	msg := message.New(topics.Score, ev)
	s.signer.Gossip(msg, s.ID())
	return nil
}
