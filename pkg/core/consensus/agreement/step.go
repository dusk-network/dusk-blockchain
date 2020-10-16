package agreement

import (
	"context"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
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
}

// New creates a round-specific agreement step
func New(e *consensus.Emitter) *Loop {
	return &Loop{
		Emitter: e,
	}
}

// GetControlFn is a factory method for the ControlFn. It makes it possible for
// the consensus loop to mock the agreement
func (s *Loop) GetControlFn() consensus.ControlFn {
	return s.Run
}

// Run the agreement step loop
func (s *Loop) Run(ctx context.Context, roundQueue *consensus.Queue, agreementChan <-chan message.Message, r consensus.RoundUpdate) {
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
		go collectEvent(h, acc, ev.Payload().(message.Agreement))
	}

	for {
		select {
		case m := <-agreementChan:
			if s.shouldCollectNow(m, r.Round, roundQueue) {
				msg := m.Payload().(message.Agreement)
				go collectEvent(h, acc, msg)
			}
		case evs := <-acc.CollectedVotesChan:
			lg.
				WithField("round", r.Round).
				WithField("step", evs[0].State().Step).
				Debugln("quorum reached")

			// Send the Agreement to the Certificate Collector within the Chain
			if err := s.sendCertificate(h, evs[0]); err != nil {
				panic(err)
			}
			return

		case <-ctx.Done():
			// finalize the worker pool
			return
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

func collectEvent(h *handler, accumulator *Accumulator, a message.Agreement) {
	hdr := a.State()
	if !h.IsMember(hdr.PubKeyBLS, hdr.Round, hdr.Step) {
		return
	}

	accumulator.Process(a)
}

func (s *Loop) sendCertificate(h *handler, ag message.Agreement) error {
	keys, err := h.getVoterKeys(ag)
	if err != nil {
		return err
	}
	cert := message.NewCertificate(ag, keys)
	msg := message.New(topics.Certificate, cert)
	_ = s.EventBus.Publish(topics.Certificate, msg)
	return nil
}
