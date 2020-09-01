package selection

import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
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

	scoreID   uint32
	scoreName string

	eventPlayer consensus.EventPlayer
	signer      consensus.Signer
	ctx         context.Context
	provisioner transactions.Provisioner
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
func NewComponent(ctx context.Context, publisher eventbus.Publisher, timeout time.Duration, provisioner transactions.Provisioner) *Selector {
	selector := &Selector{
		timeout:     timeout,
		publisher:   publisher,
		bestEvent:   message.EmptyScore(),
		ctx:         ctx,
		provisioner: provisioner,
	}
	CUSTOM_SELECTOR_TIMEOUT := os.Getenv("CUSTOM_SELECTOR_TIMEOUT")
	if CUSTOM_SELECTOR_TIMEOUT != "" {
		customTimeout, err := strconv.Atoi(CUSTOM_SELECTOR_TIMEOUT)
		if err == nil {
			log.
				WithField("customTimeout", customTimeout).
				Info("selector will set a custom timeout")
			selector.timeout = time.Duration(customTimeout) * time.Second
		} else {
			log.
				WithError(err).
				WithField("customTimeout", customTimeout).
				Error("selector could not set a custom timeout")
		}
	}

	return selector
}

// Initialize the Selector, by creating the handler and returning the needed Listeners.
// Implements consensus.Component.
func (s *Selector) Initialize(eventPlayer consensus.EventPlayer, signer consensus.Signer, r consensus.RoundUpdate) []consensus.TopicListener {
	s.eventPlayer = eventPlayer
	s.signer = signer
	s.handler = NewScoreHandler(s.provisioner)
	s.timer = &timer{s: s}

	scoreSubscriber := consensus.TopicListener{
		Topic:    topics.Score,
		Listener: consensus.NewSimpleListener(s.CollectScoreEvent, consensus.LowPriority, false),
	}
	s.scoreID = scoreSubscriber.ID()
	s.scoreName = "selection/Selector"

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

// Name returns the Name of the Score message Listener.
// Implements consensus.Component
func (s *Selector) Name() string {
	return s.scoreName
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
	h := packet.State()

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

	if err := s.handler.Verify(s.ctx, h.Round, h.Step, score); err != nil {
		return err
	}

	// Tell the candidate broker to allow a candidate block with this
	// hash through.
	msg := message.New(topics.Score, score)
	errList := s.publisher.Publish(topics.ValidCandidateHash, msg)
	diagnostics.LogPublishErrors("selection/selector.go, CollectScoreEvent, topics.ValidCandidateHash, topics.Score", errList)

	if err := s.signer.Gossip(msg, s.ID()); err != nil {
		lg.
			WithError(err).
			WithField("step", h.Step).
			WithField("round", h.Round).
			Error("CollectScoreEvent, failed to gossip")
		return err
	}

	lg.
		WithField("step", h.Step).
		WithField("round", h.Round).
		WithFields(log.Fields{
			"new_best": score.Score,
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
	s.lock.RLock()
	defer s.lock.RUnlock()
	s.timer.start(s.timeout)
}

// IncreaseTimeOut increases the timeout after a failed selection
func (s *Selector) IncreaseTimeOut() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.timeout = s.timeout * 2
	if s.timeout > 60*time.Second {
		lg.
			WithField("step", s.bestEvent.State().Step).
			WithField("round", s.bestEvent.State().Round).
			WithField("timeout", s.timeout).
			Error("max_timeout_reached")
		s.timeout = 60 * time.Second
	}
	lg.
		WithField("step", s.bestEvent.State().Step).
		WithField("round", s.bestEvent.State().Round).
		WithField("timeout", s.timeout).
		Trace("increase_timeout")
}

//nolint:unparam
func (s *Selector) sendBestEvent() error {
	var bestEvent consensus.InternalPacket
	s.eventPlayer.Pause(s.scoreID)
	s.lock.RLock()
	bestEvent = s.bestEvent
	s.lock.RUnlock()

	// If we had no best event, we should send an empty hash
	if bestEvent.(message.Score).IsEmpty() {
		bestEvent = s.signer.Compose(emptyScoreFactory{})
	}

	msg := message.New(topics.BestScore, bestEvent)
	err := s.signer.SendInternally(topics.BestScore, msg, s.ID())
	if err != nil {
		lg.WithError(err).Error("sendBestEvent, failed to SendInternally the BestScore")
	}
	s.handler.LowerThreshold()
	s.IncreaseTimeOut()
	return nil
}
