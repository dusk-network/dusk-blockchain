package firststep

import (
	"bytes"
	"context"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "first step reduction")

// Phase is the implementation of the Selection step component
type Phase struct {
	*reduction.Reduction

	handler    *reduction.Handler
	aggregator *reduction.Aggregator

	selectionResult message.Score

	next consensus.Phase
}

// New creates and launches the component which responsibility is to reduce the
// candidates gathered as winner of the selection of all nodes in the committee
// and reduce them to just one candidate obtaining 64% of the committee vote
func New(next consensus.Phase, e *consensus.Emitter, timeOut time.Duration) *Phase {
	return &Phase{
		Reduction: &reduction.Reduction{Emitter: e, TimeOut: timeOut},
		next:      next,
	}
}

// Fn passes to this reduction step the best score collected during selection
func (p *Phase) Fn(re consensus.InternalPacket) consensus.PhaseFn {
	p.selectionResult = re.(message.Score)
	return p.Run
}

// Run the first reduction step until either there is a timeout, we reach 64%
// of votes, or we experience an unrecoverable error
func (p *Phase) Run(ctx context.Context, queue *consensus.Queue, evChan chan message.Message, r consensus.RoundUpdate, step uint8) (consensus.PhaseFn, error) {
	p.handler = reduction.NewHandler(p.Keys, r.P)

	// first we send our own Selection
	if p.handler.AmMember(r.Round, step) {
		if err := p.SendReduction(r.Round, step, p.selectionResult.State().BlockHash); err != nil {
			// in case of error we need to tell the consensus loop as we cannot
			// really recover from here
			return nil, err
		}
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
			sv, err := p.collectReduction(rMsg, r.Round, step)
			if err != nil {
				return nil, err
			}

			if sv != nil {
				return p.next.Fn(*sv), nil
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

				sv, err := p.collectReduction(rMsg, r.Round, step)
				if err != nil {
					return nil, err
				}

				if sv != nil {
					// preventing timeout leakage
					go func() {
						<-timeoutChan
					}()
					return p.next.Fn(*sv), nil
				}
			}

		case <-timeoutChan:
			// in case of timeout we proceed in the consensus with an empty hash
			sv := p.createStepVoteMessage(reduction.EmptyResult, r.Round, step)
			return p.next.Fn(*sv), nil

		case <-ctx.Done():
			// preventing timeout leakage
			go func() {
				<-timeoutChan
			}()
			return nil, nil
		}
	}
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
	}).Debugln("received_event")

	result, err := p.aggregator.CollectVote(r)
	if err != nil {
		return nil, err
	}

	// if the votes converged for an empty hash we invoke halt with no
	// StepVotes
	if bytes.Equal(hdr.BlockHash, reduction.EmptyHash[:]) {
		return p.createStepVoteMessage(reduction.EmptyResult, round, step), nil
	}

	if err := p.verifyCandidateBlock(hdr.BlockHash); err != nil {
		log.
			WithError(err).
			WithField("round", hdr.Round).
			WithField("step", hdr.Step).
			Error("firststep_verifyCandidateBlock the candidate block failed")
		return p.createStepVoteMessage(reduction.EmptyResult, round, step), nil
	}

	return p.createStepVoteMessage(result, round, step), nil
}

func (p *Phase) createStepVoteMessage(r *reduction.Result, round uint64, step uint8) *message.StepVotesMsg {
	if r == nil {
		return nil
	}

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

func (p *Phase) verifyCandidateBlock(blockHash []byte) error {
	// Fetch the candidate block first.

	params := new(bytes.Buffer)
	_ = encoding.Write256(params, blockHash)
	_ = encoding.WriteBool(params, true)

	req := rpcbus.NewRequest(*params)
	timeoutGetCandidate := time.Duration(config.Get().Timeout.TimeoutGetCandidate) * time.Second
	resp, err := p.RPCBus.Call(topics.GetCandidate, req, timeoutGetCandidate)
	if err != nil {
		log.
			WithError(err).
			WithFields(log.Fields{
				"process": "reduction",
			}).Error("firststep, fetching the candidate block failed")
		return err
	}
	cm := resp.(message.Candidate)

	// If our result was not a zero value hash, we should first verify it
	// before voting on it again
	if !bytes.Equal(blockHash, reduction.EmptyHash[:]) {
		req := rpcbus.NewRequest(cm)
		timeoutVerifyCandidateBlock := time.Duration(config.Get().Timeout.TimeoutVerifyCandidateBlock) * time.Second
		if _, err := p.RPCBus.Call(topics.VerifyCandidateBlock, req, timeoutVerifyCandidateBlock); err != nil {
			log.
				WithError(err).
				WithFields(log.Fields{
					"process": "reduction",
				}).Error("firststep, verifying the candidate block failed")
			return err
		}
	}

	return nil
}
