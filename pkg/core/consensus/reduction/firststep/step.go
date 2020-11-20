package firststep

import (
	"bytes"
	"context"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "first step reduction")

func getLog(r uint64, s uint8) *log.Entry {
	return lg.WithFields(log.Fields{
		"round": r,
		"step":  s,
	})
}

// Phase is the implementation of the Selection step component
type Phase struct {
	*reduction.Reduction

	db database.DB

	handler    *reduction.Handler
	aggregator *reduction.Aggregator

	selectionResult message.Score

	verifyFn  consensus.CandidateVerificationFunc
	requestor *candidate.Requestor

	next consensus.Phase
}

// New creates and launches the component which responsibility is to reduce the
// candidates gathered as winner of the selection of all nodes in the committee
// and reduce them to just one candidate obtaining 64% of the committee vote
func New(next consensus.Phase, e *consensus.Emitter, verifyFn consensus.CandidateVerificationFunc, timeOut time.Duration, db database.DB, requestor *candidate.Requestor) *Phase {
	return &Phase{
		Reduction: &reduction.Reduction{Emitter: e, TimeOut: timeOut},
		verifyFn:  verifyFn,
		next:      next,
		db:        db,
		requestor: requestor,
	}
}

// String returns the reduction
func (p *Phase) String() string {
	return "reduction-first-step"
}

// Initialize passes to this reduction step the best score collected during selection
func (p *Phase) Initialize(re consensus.InternalPacket) consensus.PhaseFn {
	p.selectionResult = re.(message.Score)
	return p
}

// Run the first reduction step until either there is a timeout, we reach 64%
// of votes, or we experience an unrecoverable error
func (p *Phase) Run(ctx context.Context, queue *consensus.Queue, evChan chan message.Message, r consensus.RoundUpdate, step uint8) consensus.PhaseFn {
	tlog := getLog(r.Round, step)
	tlog.Traceln("starting first reduction step")

	defer func() {
		tlog.Traceln("ending first reduction step")
	}()

	p.handler = reduction.NewHandler(p.Keys, r.P)

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
			if sv := p.collectReduction(rMsg, r.Round, step); sv != nil {
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

				sv := p.collectReduction(rMsg, r.Round, step)
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

func (p *Phase) collectReduction(r message.Reduction, round uint64, step uint8) *message.StepVotesMsg {
	if err := p.handler.VerifySignature(r); err != nil {
		lg.
			WithError(err).
			WithField("round", r.State().Round).
			WithField("step", r.State().Step).
			WithField("hash", util.StringifyBytes(r.State().BlockHash)).
			Warn("error in verifying reduction, message discarded")
		return nil
	}

	hdr := r.State()
	lg.WithFields(log.Fields{
		"round": hdr.Round,
		"step":  hdr.Step,
		//"sender": hex.EncodeToString(hdr.Sender()),
		//"hash":   hex.EncodeToString(hdr.BlockHash),
	}).Debugln("received_event")

	result := p.aggregator.CollectVote(r)
	if result == nil {
		return nil
	}

	// if the votes converged for an empty hash we invoke halt with no
	// StepVotes
	if bytes.Equal(hdr.BlockHash, reduction.EmptyHash[:]) {
		return p.createStepVoteMessage(reduction.EmptyResult, round, step)
	}

	if !bytes.Equal(hdr.BlockHash, p.selectionResult.Candidate.Block.Header.Hash) {
		var err error
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(2*time.Second))
		// Ensure we release the resources associated to this context.
		defer cancel()
		p.selectionResult.Candidate, err = p.requestor.RequestCandidate(ctx, hdr.BlockHash)
		if err != nil {
			log.
				WithError(err).
				WithField("round", hdr.Round).
				WithField("step", hdr.Step).
				Error("firststep_fetchCandidateBlock failed")
			return p.createStepVoteMessage(reduction.EmptyResult, round, step)
		}

		// Store candidate for later use
		if err := p.storeCandidate(p.selectionResult.Candidate); err != nil {
			log.WithError(err).
				WithField("round", hdr.Round).
				WithField("step", hdr.Step).
				Error("firststep_storeCandidate failed")
			panic(err)
		}
	}

	if err := p.verifyFn(*p.selectionResult.Candidate.Block); err != nil {
		log.
			WithError(err).
			WithField("round", hdr.Round).
			WithField("step", hdr.Step).
			Error("firststep_verifyCandidateBlock failed")
		return p.createStepVoteMessage(reduction.EmptyResult, round, step)
	}

	return p.createStepVoteMessage(result, round, step)
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
			PubKeyBLS: p.Keys.BLSPubKeyBytes,
		},
		StepVotes: r.SV,
	}
}

func (p *Phase) storeCandidate(cm message.Candidate) error {
	return p.db.Update(func(t database.Transaction) error {
		return t.StoreCandidateMessage(cm)
	})
}
