package selection

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "selector")
var emptyScore [32]byte

// Phase is the implementation of the Selection step component
type Phase struct {
	publisher eventbus.Publisher
	handler   Handler
	bestEvent message.Score

	timeout time.Duration

	provisioner transactions.Provisioner
	next        consensus.Phase
	keys        key.Keys
}

// New creates and launches the component which responsibility is to validate
// and select the best score among the blind bidders. The component publishes under
// the topic BestScoreTopic
func New(next consensus.Phase, e *consensus.Emitter, timeout time.Duration) *Phase {
	selector := &Phase{
		timeout:     timeout,
		publisher:   e.EventBus,
		bestEvent:   message.EmptyScore(),
		provisioner: e.Proxy.Provisioner(),
		keys:        e.Keys,
		next:        next,
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

// Fn returns the Phase state function for the next phase, and initializes it
// with the result from this phase
func (p *Phase) Fn(_ consensus.InternalPacket) consensus.PhaseFn {
	return p.Run
}

// Run executes the logic for this phase
// In this case the selection listens to new Score/Candidate messages
func (p *Phase) Run(ctx context.Context, queue *consensus.Queue, evChan chan message.Message, r consensus.RoundUpdate, step uint8) (consensus.PhaseFn, error) {
	p.handler = NewScoreHandler(p.provisioner)
	timeoutChan := time.After(p.timeout)
	for _, ev := range queue.GetEvents(r.Round, step) {
		if ev.Category() == topics.Score {
			p.collectScore(ctx, ev.Payload().(message.Score))
		}
	}

	for {
		select {
		case ev := <-evChan:
			if shouldProcess(ev, r.Round, queue) {
				p.collectScore(ctx, ev.Payload().(message.Score))
			}

		case <-timeoutChan:
			phase := p.endSelection(r.Round, step)
			return phase, nil
		case <-ctx.Done():
			// preventing timeout leakage
			go func() {
				<-timeoutChan
			}()
			return nil, nil
		}
	}
}

func (p *Phase) endSelection(round uint64, step uint8) consensus.PhaseFn {

	defer func() {
		p.handler.LowerThreshold()
		p.increaseTimeOut()
	}()

	if p.bestEvent.IsEmpty() {
		hdr := header.Header{
			Round:     round,
			Step:      step,
			PubKeyBLS: p.keys.BLSPubKeyBytes,
			BlockHash: emptyScore[:],
		}
		return p.next.Fn(message.EmptyScoreProposal(hdr))
	}

	return p.next.Fn(p.bestEvent)
}

func (p *Phase) collectScore(ctx context.Context, sc message.Score) {
	// Sanity-check the candidate message
	if err := candidate.ValidateCandidate(sc.Candidate); err != nil {
		lg.Warn("Invalid candidate message")
		return
	}

	// Publish internally topics.Candidate with bestEvent(highest score candidate block)
	msg := message.New(topics.Candidate, sc.Candidate)
	p.publisher.Publish(topics.Candidate, msg)

	// Only check for priority if we already have a best event
	if !p.bestEvent.IsEmpty() {
		if p.handler.Priority(p.bestEvent, sc) {
			// if the current best score has priority, we return
			return
		}
	}

	h := sc.State()
	if err := p.handler.Verify(ctx, h.Round, h.Step, sc); err != nil {
		lg.WithError(err).Warn("Invalid score message")
		return
	}

	lg.
		WithField("step", h.Step).
		WithField("round", h.Round).
		WithFields(log.Fields{
			"new_best": sc.Score,
		}).Debugln("swapping best score")

	p.bestEvent = sc
}

// increaseTimeOut increases the timeout after a failed selection
func (p *Phase) increaseTimeOut() {
	p.timeout = p.timeout * 2
	if p.timeout > 60*time.Second {
		lg.
			WithField("step", p.bestEvent.State().Step).
			WithField("round", p.bestEvent.State().Round).
			WithField("timeout", p.timeout).
			Error("max_timeout_reached")
		p.timeout = 60 * time.Second
	}
	lg.
		WithField("step", p.bestEvent.State().Step).
		WithField("round", p.bestEvent.State().Round).
		WithField("timeout", p.timeout).
		Trace("increase_timeout")
}

func shouldProcess(a message.Message, round uint64, queue *consensus.Queue) bool {
	msg := a.Payload().(consensus.InternalPacket)
	hdr := msg.State()

	if !check(a, round, queue) {
		return false
	}

	if a.Category() != topics.Score {
		queue.PutEvent(hdr.Round, hdr.Step, a)
		return false
	}

	return true
}

func check(a message.Message, round uint64, queue *consensus.Queue) bool {
	msg := a.Payload().(consensus.InternalPacket)
	hdr := msg.State()
	if hdr.Round < round {
		lg.
			WithFields(log.Fields{
				"topic":             "Agreement",
				"round":             hdr.Round,
				"coordinator_round": round,
			}).
			Debugln("discarding obsolete agreement")
		return false
	}

	if hdr.Round > round {
		lg.
			WithFields(log.Fields{
				"topic":             "Agreement",
				"round":             hdr.Round,
				"coordinator_round": round,
			}).
			Debugln("storing future round for later")
		queue.PutEvent(hdr.Round, hdr.Step, a)
		return false
	}

	return true
}
