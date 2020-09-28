package agreement

import (
	"context"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "agreement")

// WorkerAmount sets the number of concurrent workers to concurrently verify
// Agreement messages
var WorkerAmount = 4

// Loop is the struct holding the state of the Agreement phase which does not
// change during the consensus loop
type Loop struct {
	*consensus.Emitter
	handler     *handler
	accumulator *Accumulator
}

// New creates a round-specific agreement step
func New(p user.Provisioners, e *consensus.Emitter) *Loop {
	handler := NewHandler(e.Keys, p)
	return &Loop{
		Emitter:     e,
		handler:     handler,
		accumulator: newAccumulator(handler, WorkerAmount),
	}
}

// Run the agreement step loop
func Run(ctx context.Context, roundQueue *consensus.Queue, agreementChan <-chan message.Message, r consensus.RoundUpdate, e *consensus.Emitter) error {
	// creating the step for this round
	s := New(r.P, e)

	// deferring queue cleanup at the end of the execution of this round
	defer func() {
		roundQueue.Clear(r.Round)
		s.accumulator.Stop()
	}()

	evs := roundQueue.Flush(r.Round)
	for _, ev := range evs {
		go s.collectEvent(ev.Payload().(message.Agreement))
	}

	for {
		select {
		case m := <-agreementChan:
			if s.shouldCollectNow(m, r.Round, roundQueue) {
				msg := m.Payload().(message.Agreement)
				go s.collectEvent(msg)
			}
		case evs := <-s.accumulator.CollectedVotesChan:
			lg.
				WithField("round", r.Round).
				WithField("step", evs[0].State().Step).
				Debugln("quorum reached")

			// Send the Agreement to the Certificate Collector within the Chain
			if err := s.sendCertificate(evs[0]); err != nil {
				return err
			}

			// nothing to finalize
			return nil
		case <-ctx.Done():
			// finalize the worker pool
			return nil
		}
	}
}

// TODO: consider adding a deadline for the Agreements collection and monitor
// if we get many future events (in which case we might want to return an error
// to trigger a synchronization)
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

// shouldProcess checks if the sender was in the committee
func (s *Loop) shouldProcess(hdr header.Header) bool {
	return !s.handler.IsMember(hdr.PubKeyBLS, hdr.Round, hdr.Step)
}

func (s *Loop) collectEvent(a message.Agreement) {
	if !s.shouldProcess(a.State()) {
		return
	}

	s.accumulator.Process(a)
}

func (s *Loop) sendCertificate(ag message.Agreement) error {
	keys, err := s.handler.getVoterKeys(ag)
	if err != nil {
		return err
	}
	cert := message.NewCertificate(ag, keys)
	msg := message.New(topics.Certificate, cert)
	_ = s.EventBus.Publish(topics.Certificate, msg)
	return nil
}
