// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package selection

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/blockgenerator"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "selector")

// Phase is the implementation of the Selection step component.
type Phase struct {
	*consensus.Emitter
	handler   Handler
	bestEvent message.Score

	timeout time.Duration

	provisioner transactions.Provisioner
	next        consensus.Phase
	keys        key.Keys

	g blockgenerator.BlockGenerator

	db database.DB
}

// New creates and launches the component which responsibility is to validate
// and select the best score among the blind bidders. The component publishes under
// the topic BestScoreTopic.
func New(next consensus.Phase, g blockgenerator.BlockGenerator, e *consensus.Emitter, timeout time.Duration, db database.DB) *Phase {
	selector := &Phase{
		Emitter:     e,
		timeout:     timeout,
		bestEvent:   message.EmptyScore(),
		provisioner: e.Proxy.Provisioner(),
		keys:        e.Keys,
		g:           g,
		db:          db,

		next: next,
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

// Initialize returns the Phase state function for the next phase, and initializes it
// with the result from this phase.
func (p *Phase) Initialize(_ consensus.InternalPacket) consensus.PhaseFn {
	return p
}

func (p *Phase) generateCandidate(ctx context.Context, round consensus.RoundUpdate, step uint8, internalScoreChan chan<- message.Message) {
	score := p.g.Generate(ctx, round, step)
	if score.IsEmpty() {
		return
	}

	scr, err := p.g.GenerateCandidateMessage(ctx, score, round, step)
	if err != nil {
		lg.WithError(err).Errorln("candidate block generation failed")
		return
	}

	log.
		WithField("step", step).
		WithField("round", round.Round).
		Debugln("sending score internally")

	// communicate our own score to the selection
	internalScoreChan <- message.New(topics.Score, *scr)
}

// String as required by the interface PhaseFn.
func (p *Phase) String() string {
	return "selection"
}

// Run executes the logic for this phase.
// In this case the selection listens to new Score/Candidate messages.
func (p *Phase) Run(parentCtx context.Context, queue *consensus.Queue, evChan chan message.Message, r consensus.RoundUpdate, step uint8) consensus.PhaseFn {
	ctx, cancel := context.WithCancel(parentCtx)
	// this makes sure that the internal score channel gets canceled
	defer cancel()

	// channel for the blockgenerator to communicate a score message as soon as
	// it gets generated
	internalScoreChan := make(chan message.Message, 1)

	// Since the generator can be nil in the case of the node not being a
	// block generator, let's check to see first if we actually have something.
	if p.g != nil {
		go p.generateCandidate(ctx, r, step, internalScoreChan)
	}

	p.handler = NewScoreHandler(p.provisioner)
	timeoutChan := time.After(p.timeout)

	for _, ev := range queue.GetEvents(r.Round, step) {
		if ev.Category() == topics.Score {
			p.collectScore(ctx, ev.Payload().(message.Score), ev.Header())
		}
	}

	for {
		select {
		case internalScoreResult := <-internalScoreChan:
			header := internalScoreResult.Header()
			if header == nil {
				header = []byte{config.KadcastInitialHeight}
			}

			p.collectScore(ctx, internalScoreResult.Payload().(message.Score), header)
		case ev := <-evChan:
			if shouldProcess(ev, r.Round, step, queue) {
				p.collectScore(ctx, ev.Payload().(message.Score), ev.Header())
			}

		case <-timeoutChan:
			return p.endSelection(r.Round, step)
		case <-ctx.Done():
			// preventing timeout leakage
			go func() {
				<-timeoutChan
			}()
			return nil
		}
	}
}

func (p *Phase) endSelection(_ uint64, _ uint8) consensus.PhaseFn {
	defer func() {
		p.handler.LowerThreshold()
		p.increaseTimeOut()
	}()

	if p.bestEvent.IsEmpty() {
		//TODO: check if this is required
		//hdr := header.Header{
		//	Round:     round,
		//	Step:      step,
		//	PubKeyBLS: p.keys.BLSPubKeyBytes,
		//	BlockHash: emptyScore[:],
		//}
		log.Debug("endSelection, p.next.Fn(message.EmptyScore())")
		return p.next.Initialize(message.EmptyScore())
	}

	log.Debug("endSelection, p.next.Fn(p.bestEvent)")

	e := p.bestEvent
	p.bestEvent = message.EmptyScore()
	return p.next.Initialize(e)
}

func (p *Phase) collectScore(ctx context.Context, sc message.Score, msgHeader []byte) {
	// Sanity-check the candidate message
	if err := candidate.ValidateCandidate(sc.Candidate); err != nil {
		lg.Warn("Invalid candidate message")
		return
	}

	// Publish internally topics.Candidate with bestEvent(highest score candidate block)
	if err := p.db.Update(func(t database.Transaction) error {
		return t.StoreCandidateMessage(sc.Candidate)
	}); err != nil {
		lg.WithError(err).Errorln("could not store candidate")
	}

	// Only check for priority if we already have a best event
	if !p.bestEvent.IsEmpty() {
		if p.handler.Priority(p.bestEvent, sc) {
			// if the current best score has priority, we return
			return
		}
	}

	header := sc.State()
	if err := p.handler.Verify(ctx, header.Round, header.Step, sc); err != nil {
		lg.WithError(err).Warn("Invalid score message")
		return
	}

	lg.WithField("step", header.Step).
		WithField("round", header.Round).Debugln("publishing best score")

	// Once the event is verified, and has passed all preliminary checks,
	// we can republish it to the network.

	if err := p.Republish(message.New(topics.Score, sc), msgHeader); err != nil {
		lg.WithError(err).WithField("kadcast_enabled", config.Get().Kadcast.Enabled).
			Error("could not republish score event")
	}

	lg.
		WithField("step", header.Step).
		WithField("round", header.Round).
		WithFields(log.Fields{
			"new_best": sc.Score,
		}).Debugln("swapping best score")

	p.bestEvent = sc
}

// increaseTimeOut increases the timeout after a failed selection.
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

func shouldProcess(m message.Message, round uint64, step uint8, queue *consensus.Queue) bool {
	msg := m.Payload().(consensus.InternalPacket)
	hdr := msg.State()

	cmp := hdr.CompareRoundAndStep(round, step)
	if cmp == header.Before {
		lg.
			WithFields(log.Fields{
				"topic":          m.Category(),
				"round":          hdr.Round,
				"step":           hdr.Step,
				"expected round": round,
			}).
			Debugln("discarding obsolete event")
		return false
	}

	if cmp == header.After {
		lg.
			WithFields(log.Fields{
				"topic":          m.Category(),
				"round":          hdr.Round,
				"step":           hdr.Step,
				"expected round": round,
			}).
			Debugln("storing future event for later")
		queue.PutEvent(hdr.Round, hdr.Step, m)

		return false
	}

	if m.Category() != topics.Score {
		lg.
			WithFields(log.Fields{
				"topic": m.Category(),
				"round": hdr.Round,
				"step":  hdr.Step,
			}).
			Warnln("message not topics.Score")
		return false
	}

	return true
}
