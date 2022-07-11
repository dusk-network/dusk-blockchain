// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package selection

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
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
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "consensus").WithField("phase", "selector")

var (
	errInvalidHash       = errors.New("candidate hash is different from message.header hash")
	errNotBlockGenerator = errors.New("newblock msg is not signed by a block generator")
)

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
// and select a Provisioner to propagate NewBlock message.
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
// In this case the selection listens to NewBlock messages.
func (p *Phase) Run(parentCtx context.Context, queue *consensus.Queue, evChan chan message.Message, r consensus.RoundUpdate, step uint8) consensus.PhaseFn {
	ctx, cancel := context.WithCancel(parentCtx)

	defer func() {
		p.increaseTimeOut()
		cancel()
	}()

	p.handler = NewHandler(p.Keys, r.P, r.Seed)

	isMember := p.handler.AmMember(r.Round, step)

	if log.GetLevel() >= logrus.DebugLevel {
		log := consensus.WithFields(r.Round, step, "selection_init",
			nil, p.Keys.BLSPubKey, &p.handler.Committees[step], nil, &r.P)
		log.WithField("is_member", isMember).Debug()
	}

	if isMember {
		scr, err := p.g.GenerateCandidateMessage(ctx, r, step)
		if err != nil {
			lg.WithError(err).Errorln("candidate block generation failed")
		} else {
			if log.GetLevel() >= logrus.DebugLevel {
				consensus.WithFields(r.Round, step, "candidate_generated",
					scr.State().BlockHash, p.Keys.BLSPubKey, nil, nil, nil).Debug()
			}

			buf := message.NewWithHeader(topics.NewBlock, *scr, []byte{config.KadcastInitialHeight})
			evChan <- buf
		}
	}

	for _, ev := range queue.GetEvents(r.Round, step) {
		if ev.Category() == topics.NewBlock {
			evChan <- ev
		}
	}

	timeoutChan := time.After(p.timeout)

	for {
		select {
		case ev := <-evChan:
			if shouldProcess(ev, r.Round, step, queue) {
				b := ev.Payload().(message.NewBlock)
				if err := p.collectNewBlock(b, ev.Header()); err != nil {
					continue
				}

				go func() {
					<-timeoutChan
				}()

				return p.endSelection(b)
			}
		case <-timeoutChan:
			return p.endSelection(message.EmptyNewBlock())
		case <-ctx.Done():
			// preventing timeout leakage
			go func() {
				<-timeoutChan
			}()
			return nil
		}
	}
}

func (p *Phase) endSelection(result message.NewBlock) consensus.PhaseFn {
	return p.next.Initialize(result)
}

// verifyNewBlock executes a set of check points to ensure the hash of the
// candidate block has been signed by the single committee member of selection
// step of this iteration.
func (p *Phase) verifyNewBlock(msg message.NewBlock) error {
	if !p.handler.IsMember(msg.State().PubKeyBLS, msg.State().Round, msg.State().Step) {
		return errNotBlockGenerator
	}

	// Verify message signagure
	if err := p.handler.VerifySignature(msg); err != nil {
		return err
	}

	// Sanity-check the candidate block
	if err := candidate.SanityCheckCandidate(msg.Candidate); err != nil {
		return err
	}

	// Ensure candidate block hash is equal to the BlockHash of the msg.header
	hash, err := msg.Candidate.CalculateHash()
	if err != nil {
		return err
	}

	if !bytes.Equal(msg.State().BlockHash, hash) {
		return errInvalidHash
	}

	return nil
}

func (p *Phase) collectNewBlock(msg message.NewBlock, msgHeader []byte) error {
	if err := p.verifyNewBlock(msg); err != nil {
		msg.WithFields(lg).
			WithField("seed", hex.EncodeToString(p.handler.Seed())).
			WithError(err).Error("failed to verify newblock")

		return err
	}

	// Persist Candidate Block on disk.
	if err := p.db.Update(func(t database.Transaction) error {
		return t.StoreCandidateMessage(msg.Candidate)
	}); err != nil {
		lg.WithError(err).Errorln("could not store candidate")
	}

	if log.GetLevel() >= logrus.DebugLevel {
		log := consensus.WithFields(msg.State().Round, msg.State().Step, "newblock_collected",
			msg.Candidate.Header.Hash, p.Keys.BLSPubKey, nil, nil, nil)

		log.WithField("sender", util.StringifyBytes(msg.State().PubKeyBLS)).Debug()
	}

	// Once the event is verified, and has passed all preliminary checks,
	// we can republish it to the network.
	m := message.NewWithHeader(topics.NewBlock, msg, msgHeader)
	if err := p.Republish(m); err != nil {
		lg.WithError(err).
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
			Warn("max_timeout_reached")

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
				"topic":      m.Category(),
				"msg_round":  hdr.Round,
				"msg_step":   hdr.Step,
				"curr_round": round,
				"curr_step":  step,
			}).
			Debugln("discarding obsolete event")
		return false
	}

	// Only store events up to 10 rounds into the future
	// XXX: According to protocol specs, we should abandon consensus
	// if we notice valid messages from far into the future.
	if cmp == header.After && hdr.Round-round < 10 {
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

	if m.Category() != topics.NewBlock {
		lg.
			WithFields(log.Fields{
				"topic": m.Category(),
				"round": hdr.Round,
				"step":  hdr.Step,
			}).
			Warnln("message not topics.NewBlock")
		return false
	}

	return true
}
