package secondstep

import (
	"bytes"
	"context"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "secondstep reduction")
var emptyHash [32]byte

type result struct {
	Hash []byte
	SV   message.StepVotes
}

// Phase is the implementation of the Selection step component
type Phase struct {
	*consensus.Emitter
	handler    *reduction.Handler
	aggregator *aggregator

	timeOut time.Duration

	firstStepVotesMsg message.StepVotesMsg

	next consensus.Phase
}

// New creates and launches the component which responsibility is to reduce the
// candidates gathered as winner of the selection of all nodes in the committee
// and reduce them to just one candidate obtaining 64% of the committee vote
func New(e *consensus.Emitter, timeOut time.Duration) *Phase {
	return &Phase{
		Emitter: e,
		timeOut: timeOut,
	}
}

// SetNext sets the next step to be returned at the end of this one
func (p *Phase) SetNext(next consensus.Phase) {
	p.next = next
}

// Name as dictated by the Phase interface
func (p *Phase) Name() string {
	return "secondstep_reduction"
}

// Fn passes to this reduction step the best score collected during selection
func (p *Phase) Fn(re consensus.InternalPacket) consensus.PhaseFn {
	p.firstStepVotesMsg = re.(message.StepVotesMsg)
	return p.Run
}

// Run the first reduction step until either there is a timeout, we reach 64%
// of votes, or we experience an unrecoverable error
func (p *Phase) Run(ctx context.Context, queue *consensus.Queue, evChan chan message.Message, r consensus.RoundUpdate, step uint8) (consensus.PhaseFn, error) {
	lg.
		WithField("round", r.Round).
		WithField("step", step).
		Trace("starting secondstep reduction")
	p.handler = reduction.NewHandler(p.Keys, r.P)
	// first we send our own Selection
	if p.handler.AmMember(r.Round, step) {
		if err := p.sendReduction(r.Round, step, p.firstStepVotesMsg.BlockHash); err != nil {
			// in case of error we need to tell the consensus loop as we cannot
			// really recover from here
			return nil, err
		}
	}

	timeoutChan := time.After(p.timeOut)
	p.aggregator = newAggregator(p.handler)
	for _, ev := range queue.GetEvents(r.Round, step) {
		if ev.Category() == topics.Reduction {
			rMsg := ev.Payload().(message.Reduction)
			if !p.handler.IsMember(rMsg.Sender(), r.Round, step) {
				continue
			}
			// if collectReduction returns a StepVote, it means we reached
			// consensus and can go to the next step
			svm, err := p.collectReduction(rMsg, r.Round, step)
			if err != nil {
				return nil, err
			}

			if svm == nil {
				continue
			}

			if stepVotesAreValid(&p.firstStepVotesMsg, svm) && p.handler.AmMember(r.Round, step) {
				if err := p.sendAgreement(r.Round, step, svm); err != nil {
					return nil, err
				}
			}
			return p.next.Fn(nil), nil
		}
	}

	for {
		select {
		case ev := <-evChan:
			if shouldProcess(ev, r.Round, step, queue) {
				rMsg := ev.Payload().(message.Reduction)
				if !p.handler.IsMember(rMsg.Sender(), r.Round, step) {
					continue
				}
				svm, err := p.collectReduction(rMsg, r.Round, step)
				if err != nil {
					return nil, err
				}

				if svm == nil {
					continue
				}

				go func() { // preventing timeout leakage
					<-timeoutChan
				}()

				if stepVotesAreValid(&p.firstStepVotesMsg, svm) && p.handler.AmMember(r.Round, step) {
					if err := p.sendAgreement(r.Round, step, svm); err != nil {
						return nil, err
					}
				}
				return p.next.Fn(nil), nil
			}

		case <-timeoutChan:
			// in case of timeout we increase the timeout and that's it
			p.increaseTimeout(r.Round)
			return p.next.Fn(nil), nil

		case <-ctx.Done():
			// preventing timeout leakage
			go func() {
				<-timeoutChan
			}()
			return nil, nil
		}
	}
}

func (p *Phase) sendReduction(round uint64, step uint8, hash []byte) error {
	hdr := header.Header{
		Round:     round,
		Step:      step,
		BlockHash: hash,
		PubKeyBLS: p.Keys.BLSPubKeyBytes,
	}

	sig, err := p.Sign(hdr)
	if err != nil {
		return err
	}

	red := message.NewReduction(hdr)
	red.SignedHash = sig
	_ = p.Gossip(message.New(topics.Reduction, *red))
	return nil
}

func (p *Phase) collectReduction(r message.Reduction, round uint64, step uint8) (*message.StepVotesMsg, error) {
	if err := p.handler.VerifySignature(r); err != nil {
		return nil, err
	}

	hdr := r.State()
	lg.WithFields(log.Fields{
		"round": hdr.Round,
		"step":  hdr.Step,
		//"sender": hex.EncodeToString(hdr.Sender()),
		//"hash":   hex.EncodeToString(hdr.BlockHash),
	}).Debugln("received_2nd_step_reduction")
	result, err := p.aggregator.collectVote(r)
	if err != nil {
		return nil, err
	}

	return p.createStepVoteMessage(result, round, step), nil
}

func (p *Phase) createStepVoteMessage(r *result, round uint64, step uint8) *message.StepVotesMsg {
	if r == nil {
		return nil
	}

	// quorum has been reached. However hash&votes can be empty
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

func (p *Phase) sendAgreement(round uint64, step uint8, svm *message.StepVotesMsg) error {
	hdr := header.Header{
		Round:     round,
		Step:      step,
		PubKeyBLS: p.Keys.BLSPubKeyBytes,
		BlockHash: svm.BlockHash,
	}

	sig, err := p.Sign(hdr)
	if err != nil {
		return err
	}

	// then we create the full BLS signed Agreement
	// XXX: the StepVotes are NOT signed (i.e. the message.SignAgreement is not used).
	// This exposes the Agreement to some malleability attack. Double check
	// this!!
	ev := message.NewAgreement(hdr)
	ev.SetSignature(sig)
	ev.VotesPerStep = []*message.StepVotes{
		&p.firstStepVotesMsg.StepVotes,
		&svm.StepVotes,
	}

	return p.Gossip(message.New(topics.Agreement, *ev))
}

func stepVotesAreValid(svs ...*message.StepVotesMsg) bool {
	return len(svs) == 2 &&
		!svs[0].IsEmpty() &&
		!svs[1].IsEmpty() &&
		!bytes.Equal(svs[0].BlockHash, emptyHash[:]) &&
		!bytes.Equal(svs[1].BlockHash, emptyHash[:])
}

func (p *Phase) increaseTimeout(round uint64) {
	p.timeOut = p.timeOut * 2
	if p.timeOut > 60*time.Second {
		lg.
			WithField("timeout", p.timeOut).
			WithField("round", round).
			Error("max_timeout_reached")
		p.timeOut = 60 * time.Second
	}
}

func shouldProcess(a message.Message, round uint64, step uint8, queue *consensus.Queue) bool {
	msg := a.Payload().(consensus.InternalPacket)
	hdr := msg.State()

	if !check(a, round, step, queue) {
		return false
	}

	if a.Category() != topics.Reduction {
		queue.PutEvent(hdr.Round, hdr.Step, a)
		return false
	}

	return true
}

func check(a message.Message, round uint64, step uint8, queue *consensus.Queue) bool {
	msg := a.Payload().(consensus.InternalPacket)
	hdr := msg.State()
	switch hdr.CompareRoundAndStep(round, step) {
	case header.Before:
		lg.
			WithFields(log.Fields{
				"topic":             "Agreement",
				"round":             hdr.Round,
				"coordinator_round": round,
			}).
			Debugln("discarding obsolete agreement")
		return false
	case header.After:
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
