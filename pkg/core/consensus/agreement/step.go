// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package agreement

import (
	"context"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
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
func (s *Loop) Run(ctx context.Context, roundQueue *consensus.Queue, agreementChan <-chan message.Message, r consensus.RoundUpdate) consensus.Results {
	// creating accumulator and handler
	h := NewHandler(s.Keys, r.P)
	acc := newAccumulator(h, WorkerAmount)

	// deferring queue cleanup at the end of the execution of this round
	defer func() {
		roundQueue.Clear(r.Round)
		acc.Stop()
	}()

	evs := roundQueue.Flush(r.Round)

	for _, ev := range evs {
		go collectEvent(h, acc, ev, s.Emitter)
	}

	for {
		select {
		case m := <-agreementChan:
			if s.shouldCollectNow(m, r.Round, roundQueue) {
				go collectEvent(h, acc, m, s.Emitter)
			}
		case evs := <-acc.CollectedVotesChan:
			lg.
				WithField("round", r.Round).
				WithField("step", evs[0].State().Step).
				Debugln("quorum reached")

			cert := evs[0].GenerateCertificate()
			blk, err := s.createWinningBlock(ctx, evs[0].State().BlockHash, cert)
			return consensus.Results{Blk: blk, Err: err}
		case <-ctx.Done():
			// finalize the worker pool
			return consensus.Results{Blk: block.Block{}, Err: context.Canceled}
		}
	}
}

// TODO: consider adding a deadline for the Agreements collection and monitor.
// If we get many future events (in which case we might want to return an error
// to trigger a synchronization).
func (s *Loop) shouldCollectNow(a message.Message, round uint64, queue *consensus.Queue) bool {
	hdr := a.Payload().(message.Agreement).State()
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

	// Only store events up to 10 rounds into the future
	// XXX: According to protocol specs, we should abandon consensus
	// if we notice valid messages from far into the future.
	if hdr.Round > round && hdr.Round-round < 10 {
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

func (s *Loop) createWinningBlock(ctx context.Context, hash []byte, cert *block.Certificate) (block.Block, error) {
	var cm block.Block

	err := s.db.View(func(t database.Transaction) error {
		var err error
		cm, err = t.FetchCandidateMessage(hash)
		return err
	})
	if err != nil {
		cm, err = s.requestCandidate(ctx, hash)
		if err != nil {
			return block.Block{}, err
		}
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

func collectEvent(h *handler, accumulator *Accumulator, ev message.Message, e *consensus.Emitter) {
	a := ev.Payload().(message.Agreement)

	hdr := a.State()
	if !h.IsMember(hdr.PubKeyBLS, hdr.Round, hdr.Step) {
		return
	}

	// TODO: As an optimization, ev.Header() should be used instead of always starting from KadcastInitHeader.
	m := message.NewWithHeader(topics.Agreement, a.Copy().(message.Agreement), config.KadcastInitHeader)

	// Once the event is verified, we can republish it.
	if err := e.Republish(m); err != nil {
		lg.WithError(err).Error("could not republish agreement event")
	}

	accumulator.Process(a)
}
