package firststep

import (
	"context"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "firststep reduction")
var emptyHash [32]byte
var emptyStepVotes = message.StepVotes{}

type result struct {
	Hash []byte
	SV   message.StepVotes
}

// Phase is the implementation of the Selection step component
type Phase struct {
	e          *consensus.Emitter
	handler    *reduction.Handler
	aggregator *aggregator

	timeOut time.Duration

	selectionResult message.Score

	next consensus.Phase
}

// New creates and launches the component which responsibility is to reduce the
// candidates gathered as winner of the selection of all nodes in the committee
// and reduce them to just one candidate obtaining 64% of the committee vote
func New(next consensus.Phase, e *consensus.Emitter, timeOut time.Duration) *Phase {
	return &Phase{
		e:       e,
		next:    next,
		timeOut: timeOut,
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
	p.handler = reduction.NewHandler(p.e.Keys, r.P)
	// first we send our own Selection
	if p.handler.AmMember(r.Round, step) {
		if err := p.sendReduction(r.Round, step, p.selectionResult.State().BlockHash); err != nil {
			// in case of error we need to tell the consensus loop as we cannot
			// really recover from here
			return nil, err
		}
	}

	timeoutChan := time.After(p.timeOut)
	p.aggregator = newAggregator(p.handler, p.e.RPCBus)
	for _, ev := range queue.GetEvents(r.Round, step) {
		if ev.Category() == topics.Reduction {
			// if collectReduction returns a StepVote, it means we reached
			// consensus and can go to the next step
			sv, err := p.collectReduction(ev.Payload().(message.Reduction), r.Round, step) 
			if err != nil {
				return nil, err
			}

			if sv != nil {
				return p.next.Fn(message.New(topics.StepVotes, *sv)), nil
			}
		}
	}

	for {
		select {
		case ev := <-evChan:
			if shouldProcess(ev, r.Round, queue) {
				sv, err := p.collectReduction(ev.Payload().(message.Reduction), r.Round, step)
				if err != nil {
					return nil, err
				}

				if sv != nil {
					// preventing timeout leakage
					go func() {
						<-timeoutChan
					}()
					return p.next.Fn(message.New(topics.StepVotes, *sv)), nil
				}
			}

		case <-timeoutChan:
			// in case of timeout we proceed in the consensus with an empty hash
			sv := p.createStepVoteMessage(&result{emptyHash[:], emptyStepVotes}, r.Round, step)
			return p.next.Fn(message.New(topics.StepVotes, *sv)), nil

		case <-ctx.Done():
			// preventing timeout leakage
			go func() {
				<-timeoutChan
			}()
			return nil, nil
		}
	}

	return nil, nil
}

func (p *Phase) sendReduction(round uint64, step uint8, hash []byte) error {
	hdr := header.Header{
		Round:     round,
		Step:      step,
		BlockHash: hash,
		PubKeyBLS: p.e.Keys.BLSPubKeyBytes,
	}

	sig, err := p.e.Sign(hdr)
	if err != nil {
		return err
	}

	red := message.NewReduction(hdr)
	red.SignedHash = sig
	msg := message.New(topics.Reduction, *red)
	_ = p.e.EventBus.Publish(topics.Gossip, msg)
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
	}).Debugln("received_event")
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

	if r.SV == emptyStepVotes {
		// if we converged on an empty block hash, we increase the timeout

		p.timeOut = p.timeOut * 2
		if p.timeOut > 60*time.Second {
			lg.
				WithField("timeout", p.timeOut).
				WithField("round", round).
				Error("max_timeout_reached")
			p.timeOut = 60 * time.Second
		}
	}

	return &message.StepVotesMsg{
		Header: header.Header{
			Step:      step,
			Round:     round,
			BlockHash: r.Hash,
			PubKeyBLS: p.e.Keys.BLSPubKeyBytes,
		},
		StepVotes: r.SV,
	}
}

func shouldProcess(a message.Message, round uint64, queue *consensus.Queue) bool {
	msg := a.Payload().(consensus.InternalPacket)
	hdr := msg.State()

	if !check(a, round, queue) {
		return false
	}

	if a.Category() != topics.Score {
		queue.PutEvent(hdr.Round, hdr.Step, a)
		return false
	}

	return true
}

func check(a message.Message, round uint64, queue *consensus.Queue) bool {
	msg := a.Payload().(consensus.InternalPacket)
	hdr := msg.State()
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

	if hdr.Round > round {
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
