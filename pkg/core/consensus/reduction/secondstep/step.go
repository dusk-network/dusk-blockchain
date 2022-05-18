// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package secondstep

import (
	"bytes"
	"context"
	"encoding/hex"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "consensus").
	WithField("phase", "2nd_reduction")

func getLog(r uint64, s uint8) *log.Entry {
	return lg.WithFields(log.Fields{
		"round": r,
		"step":  s,
	})
}

// Phase is the implementation of the Selection step component.
type Phase struct {
	*reduction.Reduction
	handler    *reduction.Handler
	aggregator *reduction.Aggregator

	firstStepVotesMsg message.StepVotesMsg

	next consensus.Phase
}

// New creates and launches the component which responsibility is to reduce the
// candidates gathered as winner of the selection of all nodes in the committee
// and reduce them to just one candidate obtaining 64% of the committee vote.
// NB: we cannot push the agreement directly within the agreementChannel
// until we have a way to deduplicate it from the peer (the dupemap will not be
// notified of duplicates).
func New(e *consensus.Emitter, verifyFn consensus.CandidateVerificationFunc, timeOut time.Duration) *Phase {
	return &Phase{
		Reduction: &reduction.Reduction{
			Emitter:  e,
			TimeOut:  timeOut,
			VerifyFn: verifyFn,
		},
	}
}

// SetNext sets the next step to be returned at the end of this one.
func (p *Phase) SetNext(next consensus.Phase) {
	p.next = next
}

// String representation of this Phase.
func (p *Phase) String() string {
	return "reduction-second-step"
}

// Initialize passes to this reduction step the best score collected during selection.
func (p *Phase) Initialize(re consensus.InternalPacket) consensus.PhaseFn {
	p.firstStepVotesMsg = re.(message.StepVotesMsg)
	p.VerifiedHash = p.firstStepVotesMsg.VerifiedHash
	return p
}

// Run the first reduction step until either there is a timeout, we reach 64%
// of votes, or we experience an unrecoverable error.
func (p *Phase) Run(ctx context.Context, queue *consensus.Queue, evChan chan message.Message, r consensus.RoundUpdate, step uint8) consensus.PhaseFn {
	tlog := getLog(r.Round, step)

	defer func() {
		tlog.Traceln("ending second reduction step")
	}()

	if log.GetLevel() >= logrus.DebugLevel {
		c := p.firstStepVotesMsg.Candidate
		tlog.WithField("hash", util.StringifyBytes(c.Header.Hash)).
			Debug("initialized")
	}

	p.handler = reduction.NewHandler(p.Keys, r.P, r.Seed)
	// first we send our own Selection
	if p.handler.AmMember(r.Round, step) {
		m, _ := p.SendReduction(r.Round, step, p.firstStepVotesMsg.Candidate)

		// Queue my own vote to be registered locally
		evChan <- m
	}

	timeoutChan := time.After(p.TimeOut)
	p.aggregator = reduction.NewAggregator(p.handler)

	for _, ev := range queue.GetEvents(r.Round, step) {
		if ev.Category() == topics.Reduction {
			rMsg := ev.Payload().(message.Reduction)

			if !p.handler.IsMember(rMsg.Sender(), r.Round, step) {
				continue
			}

			// if collectReduction returns a StepVote, it means we reached
			// consensus and can go to the next step
			svm := p.collectReduction(rMsg, r.Round, step, ev.Header())
			if svm == nil {
				continue
			}

			if stepVotesAreValid(&p.firstStepVotesMsg, svm) && p.handler.AmMember(r.Round, step) {
				p.sendAgreement(r.Round, step, svm)
			}

			return p.next.Initialize(nil)
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

				svm := p.collectReduction(rMsg, r.Round, step, ev.Header())
				if svm == nil {
					continue
				}

				go func() { // preventing timeout leakage
					<-timeoutChan
				}()

				if stepVotesAreValid(&p.firstStepVotesMsg, svm) && p.handler.AmMember(r.Round, step) {
					p.sendAgreement(r.Round, step, svm)
				}

				return p.next.Initialize(nil)
			}

		case <-timeoutChan:
			// in case of timeout we increase the timeout and that's it
			p.IncreaseTimeout(r.Round)
			return p.next.Initialize(nil)

		case <-ctx.Done():
			// preventing timeout leakage
			go func() {
				<-timeoutChan
			}()

			return nil
		}
	}
}

func (p *Phase) collectReduction(r message.Reduction, round uint64, step uint8, msgHeader []byte) *message.StepVotesMsg {
	hdr := r.State()

	if err := p.handler.VerifySignature(r.Copy().(message.Reduction)); err != nil {
		lg.
			WithError(err).
			WithFields(log.Fields{
				"round":  hdr.Round,
				"step":   hdr.Step,
				"sender": util.StringifyBytes(hdr.Sender()),
				"hash":   util.StringifyBytes(hdr.BlockHash),
			}).
			Warn("signature verification error for second step reduction, discarding")
		return nil
	}

	if log.GetLevel() >= logrus.DebugLevel {
		log := consensus.WithFields(hdr.Round, hdr.Step, "2nd_reduction_collected",
			hdr.BlockHash, p.handler.BLSPubKey, nil, nil, nil)

		log.WithField("sender", util.StringifyBytes(hdr.Sender())).
			Debug("")
	}

	m := message.NewWithHeader(topics.Reduction, r.Copy().(message.Reduction), msgHeader)

	// Once the event is verified, we can republish it.
	if err := p.Emitter.Republish(m); err != nil {
		lg.WithError(err).Error("could not republish reduction event")
	}

	result := p.aggregator.CollectVote(r)

	return p.createStepVoteMessage(result, round, step)
}

func (p *Phase) createStepVoteMessage(r *reduction.Result, round uint64, step uint8) *message.StepVotesMsg {
	if r == nil {
		return nil
	}

	lg.WithFields(log.Fields{
		"round":         round,
		"step":          step,
		"result_empty?": r.IsEmpty(),
	}).Debugln("quorum reached")

	// quorum has been reached. However hash&votes can be empty
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

func (p *Phase) sendAgreement(round uint64, step uint8, svm *message.StepVotesMsg) {
	lg.WithFields(log.Fields{
		"round": round,
		"step":  step,
		"hash":  hex.EncodeToString(svm.BlockHash),
	}).Debugln("sending_agreement")

	hdr := header.Header{
		Round:     round,
		Step:      step,
		PubKeyBLS: p.Keys.BLSPubKey,
		BlockHash: svm.BlockHash,
	}

	sig, err := p.Sign(hdr)
	if err != nil {
		panic(err)
	}

	lg.WithFields(log.Fields{
		"round": round,
		"step":  step,
	}).Traceln("agreement_signed")

	// then we create the full BLS signed Agreement
	// XXX: the StepVotes are NOT signed (i.e. the message.SignAgreement is not used).
	// This exposes the Agreement to some malleability attack. Double check
	// this!!
	ev := message.NewAgreement(hdr)
	ev.VotesPerStep = []*message.StepVotes{
		&p.firstStepVotesMsg.StepVotes,
		&svm.StepVotes,
	}

	ev.SetSignature(sig)

	lg.WithFields(log.Fields{
		"round": round,
		"step":  step,
	}).Traceln("publishing_agreement_internally")

	// Publishing Agreement internally so that it's the internal Agreement process(goroutine)
	// that should register it locally and only then broadcast it.
	m := message.NewWithHeader(topics.Agreement, *ev, config.KadcastInitHeader)
	p.EventBus.Publish(topics.Agreement, m)
}

func stepVotesAreValid(svs ...*message.StepVotesMsg) bool {
	return len(svs) == 2 &&
		!svs[0].IsEmpty() &&
		!svs[1].IsEmpty() &&
		!bytes.Equal(svs[0].BlockHash, block.EmptyHash[:]) &&
		!bytes.Equal(svs[1].BlockHash, block.EmptyHash[:])
}
