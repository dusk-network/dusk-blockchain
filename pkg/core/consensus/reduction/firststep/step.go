// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package firststep

import (
	"bytes"
	"context"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "consensus").
	WithField("phase", "1th_reduction")

func getLog(r uint64, s uint8) *log.Entry {
	return lg.WithFields(log.Fields{
		"round": r,
		"step":  s,
	})
}

// Phase is the implementation of the Selection step component.
type Phase struct {
	*reduction.Reduction

	db database.DB

	handler    *reduction.Handler
	aggregator *reduction.Aggregator

	selectionResult message.NewBlock

	verifyFn  consensus.CandidateVerificationFunc
	requestor *candidate.Requestor

	next consensus.Phase
}

// New creates and launches the component which responsibility is to reduce the
// candidates gathered as winner of the selection of all nodes in the committee
// and reduce them to just one candidate obtaining 64% of the committee vote.
func New(next consensus.Phase, e *consensus.Emitter, verifyFn consensus.CandidateVerificationFunc, timeOut time.Duration, db database.DB, requestor *candidate.Requestor) *Phase {
	return &Phase{
		Reduction: &reduction.Reduction{Emitter: e, TimeOut: timeOut},
		verifyFn:  verifyFn,
		next:      next,
		db:        db,
		requestor: requestor,
	}
}

// String returns the reduction.
func (p *Phase) String() string {
	return "reduction-first-step"
}

// Initialize passes to this reduction step the best score collected during selection.
func (p *Phase) Initialize(re consensus.InternalPacket) consensus.PhaseFn {
	p.selectionResult = re.(message.NewBlock)
	return p
}

// Run the first reduction step until either there is a timeout, we reach 64%
// of votes, or we experience an unrecoverable error.
func (p *Phase) Run(ctx context.Context, queue *consensus.Queue, evChan chan message.Message, r consensus.RoundUpdate, step uint8) consensus.PhaseFn {
	tlog := getLog(r.Round, step)
	tlog.Traceln("starting first reduction step")

	defer func() {
		tlog.Traceln("ending first reduction step")
	}()

	p.handler = reduction.NewHandler(p.Keys, r.P, r.Seed)

	// first we send our own Selection
	if p.handler.AmMember(r.Round, step) {
		p.SendReduction(r.Round, step, p.selectionResult.State().BlockHash)
	}

	timeoutChan := time.After(p.TimeOut)
	p.aggregator = reduction.NewAggregator(p.handler)

	for _, ev := range queue.GetEvents(r.Round, step) {
		if ev.Category() == topics.Reduction {
			rMsg := ev.Payload().(message.Reduction)

			// if the sender is no member we discard the message
			// XXX: the fact that a message from a non-committee member can end
			// up in the Queue, is a vulnerability since an attacker could
			// flood the queue with future non-committee reductions
			if !p.handler.IsMember(rMsg.Sender(), r.Round, step) {
				continue
			}

			// if collectReduction returns a StepVote, it means we reached
			// consensus and can go to the next step
			if sv := p.collectReduction(ctx, rMsg, r.Round, step); sv != nil {
				go func() {
					<-timeoutChan
				}()
				return p.next.Initialize(*sv)
			}
		}
	}

	for {
		select {
		case ev := <-evChan:
			if reduction.ShouldProcess(ev, r.Round, step, queue) {
				rMsg := ev.Payload().(message.Reduction)
				if !p.handler.IsMember(rMsg.Sender(), r.Round, step) {
					continue
				}

				sv := p.collectReduction(ctx, rMsg, r.Round, step)
				if sv != nil {
					// preventing timeout leakage
					go func() {
						<-timeoutChan
					}()
					return p.next.Initialize(*sv)
				}
			}

		case <-timeoutChan:
			// in case of timeout we proceed in the consensus with an empty hash
			sv := p.createStepVoteMessage(reduction.EmptyResult, r.Round, step)
			return p.next.Initialize(*sv)

		case <-ctx.Done():
			// preventing timeout leakage
			go func() {
				<-timeoutChan
			}()
			return nil
		}
	}
}

func (p *Phase) collectReduction(ctx context.Context, r message.Reduction, round uint64, step uint8) *message.StepVotesMsg {
	if err := p.handler.VerifySignature(r.Copy().(message.Reduction)); err != nil {
		lg.
			WithError(err).
			WithField("round", r.State().Round).
			WithField("step", r.State().Step).
			WithField("hash", util.StringifyBytes(r.State().BlockHash)).
			Warn("error in verifying reduction, message discarded")
		return nil
	}

	if log.GetLevel() >= logrus.DebugLevel {
		log := consensus.WithFields(r.State().Round, r.State().Step, "1th_reduction_collected",
			r.State().BlockHash, p.handler.BLSPubKey, nil, nil, nil)

		log.WithField("signature", util.StringifyBytes(r.SignedHash)).
			WithField("sender", util.StringifyBytes(r.Sender())).
			Debug("")
	}

	m := message.NewWithHeader(topics.Reduction, r, config.KadcastInitHeader)

	// Once the event is verified, we can republish it.
	if err := p.Emitter.Republish(m); err != nil {
		lg.WithError(err).Error("could not republish reduction event")
	}

	hdr := r.State()

	result := p.aggregator.CollectVote(r)
	if result == nil {
		return nil
	}

	// if the votes converged for an empty hash we invoke halt with no
	// StepVotes
	if bytes.Equal(hdr.BlockHash, reduction.EmptyHash[:]) {
		return p.createStepVoteMessage(reduction.EmptyResult, round, step)
	}

	// start measuring the execution time.
	st := time.Now().UnixMilli()

	if !bytes.Equal(hdr.BlockHash, p.selectionResult.Candidate.Header.Hash) {
		var err error

		p.selectionResult.Candidate, err = p.fetchCandidate(ctx, hdr.BlockHash)
		if err != nil {
			log.
				WithError(err).
				WithField("round", hdr.Round).
				WithField("step", hdr.Step).
				Error("firststep_fetchCandidateBlock failed")
			return p.createStepVoteMessage(reduction.EmptyResult, round, step)
		}
	}

	if err := p.verifyFn(p.selectionResult.Candidate); err != nil {
		log.
			WithError(err).
			WithField("round", hdr.Round).
			WithField("step", hdr.Step).
			Error("firststep_verifyCandidateBlock failed")
		return p.createStepVoteMessage(reduction.EmptyResult, round, step)
	}

	if step > 3 {
		maxDelay := config.Get().Consensus.ThrottleIterMilli
		if maxDelay == 0 {
			maxDelay = 1000
		}

		d, err := util.Throttle(st, maxDelay)
		if err == nil {
			log.WithField("step", step).WithField("sleep_for", d.String()).Trace("vst throttled")
		}
	}

	return p.createStepVoteMessage(result, round, step)
}

func (p *Phase) fetchCandidate(ctx context.Context, hash []byte) (block.Block, error) {
	// First, check to see if we have the candidate in the db.
	var cm block.Block

	err := p.db.View(func(t database.Transaction) error {
		var err error
		cm, err = t.FetchCandidateMessage(hash)
		return err
	})
	if err == nil && !cm.Equals(&block.Block{}) {
		return cm, nil
	}

	return p.requestCandidate(ctx, hash)
}

func (p *Phase) requestCandidate(ctx context.Context, hash []byte) (block.Block, error) {
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(2*time.Second))
	// Ensure we release the resources associated to this context.
	defer cancel()

	cm, err := p.requestor.RequestCandidate(ctx, hash)
	if err != nil {
		return block.Block{}, err
	}

	// Store candidate for later use
	if err := p.storeCandidate(cm); err != nil {
		panic(err)
	}

	return cm, nil
}

func (p *Phase) createStepVoteMessage(r *reduction.Result, round uint64, step uint8) *message.StepVotesMsg {
	if r.IsEmpty() {
		p.IncreaseTimeout(round)
	}

	return &message.StepVotesMsg{
		Header: header.Header{
			Step:      step,
			Round:     round,
			BlockHash: r.Hash,
			PubKeyBLS: p.Keys.BLSPubKey,
		},
		StepVotes: r.SV,
	}
}

func (p *Phase) storeCandidate(cm block.Block) error {
	return p.db.Update(func(t database.Transaction) error {
		return t.StoreCandidateMessage(cm)
	})
}
