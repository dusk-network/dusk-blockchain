// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package agreement

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/dusk-network/bls12_381-sign/go/cgo/bls"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "consensus").WithField("phase", "agreement")

// WorkerAmount sets the number of concurrent workers to concurrently verify
// Agreement messages.
var WorkerAmount = 4

// Loop is the struct holding the state of the Agreement phase which does not
// change during the consensus loop.
type Loop struct {
	*consensus.Emitter
	db        database.DB
	requestor *candidate.Requestor
}

// New creates a round-specific agreement step.
func New(e *consensus.Emitter, db database.DB, requestor *candidate.Requestor) *Loop {
	return &Loop{
		Emitter:   e,
		db:        db,
		requestor: requestor,
	}
}

// GetControlFn is a factory method for the ControlFn. It makes it possible for
// the consensus loop to mock the agreement.
func (s *Loop) GetControlFn() consensus.ControlFn {
	return s.Run
}

// Run the agreement step loop.
func (s *Loop) Run(ctx context.Context, roundQueue *consensus.Queue, agreementChan <-chan message.Message, aggrAgreementChan <-chan message.Message, r consensus.RoundUpdate) consensus.Results {
	// creating accumulator and handler
	handler := NewHandler(s.Keys, r.P, r.Seed)
	acc := newAccumulator(handler, WorkerAmount)

	// deferring queue cleanup at the end of the execution of this round
	defer func() {
		roundQueue.Clear(r.Round)
		acc.Stop()
	}()

	// Process event queue
	evs := roundQueue.Flush(r.Round)
	for _, ev := range evs {
		switch ev.Category() {
		case topics.Agreement:
			// Agreement - Verify, broadcast and add to accumulator
			go collectAgreement(handler, acc, ev, s.Emitter)
		case topics.AggrAgreement:
			// AggrAgreement - Verify, broadcast and create a block candidate
			// Process aggregated agreement
			res, err := s.processAggrAgreement(ctx, handler, ev, r, s.Emitter)
			if err != nil {
				lg.WithError(err).Errorln("failed to process aggragreement")
			} else {
				return *res // TODO: Might be wise to do a copy before returning
			}
		}
	}

	for {
		// Priority:
		// 1a - CollectedVotesChan: We give priority to our own certificate in case it gets produced in time
		// 1b - CertificateChan: Certificate sent by other nodes
		// 1c - Context.Done()
		// 2 - AgreementChan: Collect agreement messages
		//
		// The way to prioritize among select is to use the continue statement
		select {
		// 1a - CollectedVotesChan: We give priority to our own certificate in case it gets produced in time
		case evs := <-acc.CollectedVotesChan:
			return s.processCollectedVotes(ctx, handler, evs, r)
		// 1b - CertificateChan: Certificate sent by other nodes
		case cert := <-aggrAgreementChan:
			if s.shouldCollectNow(cert, r.Round, roundQueue) {
				// AggrAgreement - Verify, broadcast and create a block candidate
				// Process aggregated agreement
				if res, err := s.processAggrAgreement(ctx, handler, cert, r, s.Emitter); err != nil {
					lg.WithError(err).Errorln("failed to process aggr agreement")
				} else {
					return *res // TODO: Might be wise to do a copy before returning
				}
			}
		// 1c - Context.Done()
		case <-ctx.Done():
			// finalize the worker pool
			return consensus.Results{Blk: block.Block{}, Err: context.Canceled}
		default:
		low_priority:
			select {
			// 1a - CollectedVotesChan: We give priority to our own certificate in case it gets produced in time
			case evs := <-acc.CollectedVotesChan:
				return s.processCollectedVotes(ctx, handler, evs, r)
			// 1b - CertificateChan: Certificate sent by other nodes
			case cert := <-aggrAgreementChan:
				if s.shouldCollectNow(cert, r.Round, roundQueue) {
					// AggrAgreement - Verify, broadcast and create a block candidate
					// Process aggregated agreement
					if res, err := s.processAggrAgreement(ctx, handler, cert, r, s.Emitter); err != nil {
						lg.WithError(err).Errorln("failed to process aggr agreement")
					} else {
						return *res // TODO: Might be wise to do a copy before returning
					}
				}
			// 1c - Context.Done()
			case <-ctx.Done():
				// finalize the worker pool
				return consensus.Results{Blk: block.Block{}, Err: context.Canceled}
			// 2 - AgreementChan: Collect agreement messages
			case m := <-agreementChan:
				if s.shouldCollectNow(m, r.Round, roundQueue) {
					go collectAgreement(handler, acc, m, s.Emitter)
				}
				break low_priority // Prevents us from getting stuck in the low priority select
			}
		}
	}
}

// TODO: consider adding a deadline for the Agreements collection and monitor.
// If we get many future events (in which case we might want to return an error
// to trigger a synchronization).
func (s *Loop) shouldCollectNow(a message.Message, round uint64, queue *consensus.Queue) bool {
	var hdr header.Header
	var topicstr string

	switch a.Category() {
	case topics.Agreement:
		hdr = a.Payload().(message.Agreement).State()
		topicstr = "Agreement"
	case topics.AggrAgreement:
		hdr = a.Payload().(message.AggrAgreement).State()
		topicstr = "AggrAgreement"
	default:
		return false
	}

	if hdr.Round < round {
		lg.
			WithFields(log.Fields{
				"topic":  topicstr,
				"round":  hdr.Round,
				"curr_h": round,
			}).
			Debugln("discarding obsolete message")
		return false
	}

	// XXX: According to protocol specs, we should abandon consensus
	// if we notice valid messages from far into the future.
	if hdr.Round > round {
		if hdr.Round-round < 10 {
			// Only store events up to 10 rounds into the future
			lg.
				WithFields(log.Fields{
					"topic":  topicstr,
					"round":  hdr.Round,
					"curr_h": round,
				}).
				Debugln("storing future round for later")
			queue.PutEvent(hdr.Round, hdr.Step, a)
		} else {
			lg.
				WithFields(log.Fields{
					"topic":  topicstr,
					"round":  hdr.Round,
					"curr_h": round,
				}).
				Info("discarding too-far message")
		}

		return false
	}

	return true
}

func (s *Loop) createWinningBlock(ctx context.Context, hash []byte, cert *block.Certificate) (block.Block, error) {
	var cm block.Block

	err := s.db.View(func(t database.Transaction) error {
		var err error
		cm, err = t.FetchCandidateMessage(hash)
		return err
	})
	if err != nil {
		lg.WithField("hash", util.StringifyBytes(hash)).Info("request candidate block")

		cm, err = s.requestCandidate(ctx, hash)
		if err != nil {
			lg.WithField("hash", util.StringifyBytes(hash)).WithError(err).
				Warn("failed to receive candidate block")
			return block.Block{}, err
		}

		lg.WithField("hash", util.StringifyBytes(hash)).Info("candidate block received")
	}

	cm.Header.Certificate = cert
	return cm, nil
}

func (s *Loop) requestCandidate(ctx context.Context, hash []byte) (block.Block, error) {
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(2*time.Second))
	// Ensure we release the resources associated to this context.
	defer cancel()
	return s.requestor.RequestCandidate(ctx, hash)
}

func collectAgreement(h *handler, accumulator *Accumulator, ev message.Message, e *consensus.Emitter) {
	a := ev.Payload().(message.Agreement)

	hdr := a.State()
	if !h.IsMember(hdr.PubKeyBLS, hdr.Round, hdr.Step) {
		return
	}

	m := message.NewWithHeader(topics.Agreement, a.Copy().(message.Agreement), ev.Header())

	// Once the event is verified, we can republish it.
	if err := e.Republish(m); err != nil {
		lg.WithError(err).Error("could not republish agreement event")
	}

	accumulator.Process(a)
}

func (s *Loop) processCollectedVotes(ctx context.Context, handler *handler, evs []message.Agreement, r consensus.RoundUpdate) consensus.Results {
	lg.
		WithField("round", r.Round).
		WithField("step", evs[0].State().Step).
		WithField("stepvotes", evs[0].VotesPerStep).
		WithField("hash", util.StringifyBytes(evs[0].State().BlockHash)).
		Info("consensus_achieved")

	// Aggregate all agreement signatures into a single BLS signature
	sigs := [][]byte{}
	comm := handler.Committee(r.Round, evs[0].State().Step)
	pubs := new(sortedset.Set)

	for _, a := range evs {
		if pubs.Insert(a.State().PubKeyBLS) {
			sigs = append(sigs, a.Signature())
		}
	}

	sig := sigs[0]

	if len(sigs) > 1 {
		if sigagg, err := bls.AggregateSig(sig, sigs[1:]...); err != nil {
			lg.WithError(err).Error("could not aggregate signatures")
		} else {
			sig = sigagg
		}
	}

	// TODO: #1192 - Propagate AggrAgreement
	// Create new message
	bits := comm.Bits(*pubs)
	agAgreement := message.NewAggrAgreement(evs[0], bits, sig)

	m := message.NewWithHeader(topics.AggrAgreement, agAgreement, config.KadcastInitHeader)
	if err := s.Emitter.Republish(m); err != nil {
		lg.WithError(err).Error("could not republish aggregated agreement event")
	} else {
		lg.
			WithField("round", r.Round).
			WithField("step", agAgreement.State().Step).
			WithField("hash", util.StringifyBytes(agAgreement.State().BlockHash)).
			WithField("bitset", agAgreement.Bitset).
			Infoln("aggragreement republished")
	}

	// Create a block and return
	cert := evs[0].GenerateCertificate()

	blk, err := s.createWinningBlock(ctx, evs[0].State().BlockHash, cert)
	if err != nil {
		lg.WithError(err).Errorln("failed to create a winning block")
	}

	return consensus.Results{Blk: blk, Err: err}
}

func (s *Loop) processAggrAgreement(ctx context.Context, h *handler, msg message.Message, r consensus.RoundUpdate, e *consensus.Emitter) (*consensus.Results, error) {
	var err error
	var blk block.Block

	aggro := msg.Payload().(message.AggrAgreement)

	hdr := aggro.State()

	lg.
		WithField("round", r.Round).
		WithField("step", hdr.Step).
		WithField("hash", util.StringifyBytes(hdr.BlockHash)).
		WithField("bitset", aggro.Bitset).
		Debugln("processing aggragreement")

	// Verify certificate
	comm := h.Committee(hdr.Round, hdr.Step)

	voters := comm.Intersect(aggro.Bitset)

	subcommittee := comm.IntersectCluster(aggro.Bitset)

	allVoters := subcommittee.TotalOccurrences()
	if allVoters < h.Quorum(hdr.Round) {
		return nil, fmt.Errorf("vote set too small - %v/%v", allVoters, h.Quorum(hdr.Round))
	}
	// Aggregate keys
	apk, err := AggregatePks(&h.Provisioners, voters)
	if err != nil {
		lg.WithError(err).Errorln("could not aggregate keys")
		return nil, err
	}

	// Verify signature (AggrAgreement)
	buf := new(bytes.Buffer)

	err = header.MarshalSignableVote(buf, hdr)
	if err != nil {
		lg.WithError(err).Errorln("failed to marshal aggragreement")
		return nil, err
	}

	err = bls.Verify(apk, aggro.AggrSig, buf.Bytes())
	if err != nil {
		lg.
			WithField("round", r.Round).
			WithField("step", aggro.State().Step).
			WithField("hash", util.StringifyBytes(aggro.State().BlockHash)).
			WithField("bitset", aggro.Bitset).
			WithError(err).Errorln("failed to verify aggragreement signature")
		return nil, err
	}

	// Broadcast
	m := message.NewWithHeader(topics.AggrAgreement, aggro.Copy(), config.KadcastInitHeader)
	if err = e.Republish(m); err != nil {
		lg.WithError(err).Errorln("could not republish aggragreement event")
		return nil, err
	}

	blk, err = s.createWinningBlock(ctx, aggro.State().BlockHash, aggro.GenerateCertificate())
	if err != nil {
		lg.WithError(err).Errorln("failed to create a winning block")
		return nil, err
	}

	lg.
		WithField("round", r.Round).
		WithField("step", hdr.Step).
		WithField("hash", util.StringifyBytes(hdr.BlockHash)).
		WithField("stepvotes", aggro.VotesPerStep).
		WithField("short-circuit", true).
		Info("consensus_achieved")

	return &consensus.Results{Blk: blk}, nil
}
