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
	"github.com/dusk-network/dusk-blockchain/pkg/util"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "selector")

// Phase is the implementation of the Selection step component.
type Phase struct {
	*consensus.Emitter
	handler *Handler

	timeout time.Duration

	next consensus.Phase
	keys key.Keys

	g blockgenerator.BlockGenerator

	db database.DB
}

// New creates and launches the component which responsibility is to validate
// and select the best score among the blind bidders. The component publishes under
// the topic BestScoreTopic.
func New(next consensus.Phase, g blockgenerator.BlockGenerator, e *consensus.Emitter, timeout time.Duration, db database.DB) *Phase {
	selector := &Phase{
		Emitter: e,
		timeout: timeout,
		keys:    e.Keys,
		g:       g,
		db:      db,

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

// String as required by the interface PhaseFn.
func (p *Phase) String() string {
	return "selection"
}

// Run executes the logic for this phase.
// In this case the selection listens to new Score/Candidate messages.
func (p *Phase) Run(parentCtx context.Context, queue *consensus.Queue, evChan chan message.Message, r consensus.RoundUpdate, step uint8) consensus.PhaseFn {
	ctx, cancel := context.WithCancel(parentCtx)

	defer func() {
		p.increaseTimeOut()
		cancel()
	}()

	p.handler = NewHandler(p.Keys, r.P)

	isMember := p.handler.AmMember(r.Round, step)

	log.WithField("process", "consensus").
		WithField("round", r.Round).
		WithField("provisioners", r.P).
		WithField("step_voting_committee", p.handler.Committees[step]).
		WithField("step", step).
		WithField("is_member", isMember).
		WithField("this_provisioner", util.StringifyBytes(p.Keys.BLSPubKey)).
		WithField("node_id", config.Get().Network.Port).
		WithField("event", "selection_initialized").Debug("")

	if isMember {
		scr, err := p.g.GenerateCandidateMessage(ctx, r, step)
		if err != nil {
			lg.WithError(err).Errorln("candidate block generation failed")
		}

		log.WithField("process", "consensus").
			WithField("round", r.Round).
			WithField("hash", util.StringifyBytes(scr.State().BlockHash)).
			WithField("this_provisioner", util.StringifyBytes(p.keys.BLSPubKey)).
			WithField("step", step).
			WithField("event", "candidate_generated").Debug("")

		evChan <- message.NewWithHeader(topics.Score, *scr, []byte{config.KadcastInitialHeight})
	}

	timeoutChan := time.After(p.timeout)

	for _, ev := range queue.GetEvents(r.Round, step) {
		if ev.Category() == topics.Score {
			evChan <- ev
		}
	}

	for {
		select {
		case ev := <-evChan:
			if shouldProcess(ev, r.Round, step, queue) {
				if err := p.collectScore(ev.Payload().(message.Score), ev.Header()); err != nil {
					continue
				}

				// preventing timeout leakage
				go func() {
					<-timeoutChan
				}()
				return p.endSelection(ev.Payload().(message.Score))
			}
		case <-timeoutChan:
			return p.endSelection(message.EmptyScore())
		case <-ctx.Done():
			// preventing timeout leakage
			go func() {
				<-timeoutChan
			}()
			return nil
		}
	}
}

func (p *Phase) endSelection(result message.Score) consensus.PhaseFn {
	log.WithField("empty", result.IsEmpty()).
		Debug("endSelection")
	return p.next.Initialize(result)
}

func (p *Phase) collectScore(sc message.Score, msgHeader []byte) error {
	// Sanity-check the candidate message
	if err := candidate.ValidateCandidate(sc.Candidate); err != nil {
		lg.Warn("Invalid candidate message")
		return err
	}

	// Consensus-check the candidate message
	if err := p.handler.VerifySignature(sc); err != nil {
		lg.Warn("incorrect candidate message")
		return err
	}

	if err := p.db.Update(func(t database.Transaction) error {
		return t.StoreCandidateMessage(sc.Candidate)
	}); err != nil {
		lg.WithError(err).Errorln("could not store candidate")
	}

	// Once the event is verified, and has passed all preliminary checks,
	// we can republish it to the network.
	m := message.NewWithHeader(topics.Score, sc, msgHeader)
	if err := p.Republish(m); err != nil {
		lg.WithError(err).WithField("kadcast_enabled", config.Get().Kadcast.Enabled).
			Error("could not republish score event")
	}

	return nil
}

// increaseTimeOut increases the timeout after a failed selection.
func (p *Phase) increaseTimeOut() {
	p.timeout = p.timeout * 2
	if p.timeout > 60*time.Second {
		lg.
			WithField("timeout", p.timeout).
			Error("max_timeout_reached")

		p.timeout = 60 * time.Second
	}

	lg.
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
