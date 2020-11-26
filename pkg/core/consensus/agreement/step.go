package agreement

import (
	"context"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
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
	db           database.DB
	newBlockChan chan consensus.Results
}

// New creates a round-specific agreement step
func New(e *consensus.Emitter, db database.DB, newBlockChan chan consensus.Results) *Loop {
	return &Loop{
		Emitter:      e,
		db:           db,
		newBlockChan: newBlockChan,
	}
}

// GetControlFn is a factory method for the ControlFn. It makes it possible for
// the consensus loop to mock the agreement
func (s *Loop) GetControlFn() consensus.ControlFn {
	return s.Run
}

// Run the agreement step loop
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

			cert := evs[0].GenerateCertificate()
			blk, err := s.createWinningBlock(evs[0].State().BlockHash, cert)
			return consensus.Results{Blk: blk, Err: err}
		case r := <-s.newBlockChan:
			return r
		case <-ctx.Done():
			// finalize the worker pool
			return consensus.Results{Blk: block.Block{}, Err: context.Canceled}
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

func (s *Loop) createWinningBlock(hash []byte, cert *block.Certificate) (block.Block, error) {
	var cm block.Block
	if err := s.db.View(func(t database.Transaction) error {
		var err error
		cm, err = t.FetchCandidateMessage(hash)
		return err
	}); err != nil {
		return block.Block{}, err
	}

	cm.Header.Certificate = cert
	return cm, nil
}

func collectEvent(h *handler, accumulator *Accumulator, a message.Agreement) {
	hdr := a.State()
	if !h.IsMember(hdr.PubKeyBLS, hdr.Round, hdr.Step) {
		return
	}

	accumulator.Process(a)
}
