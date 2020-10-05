package reduction

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	log "github.com/sirupsen/logrus"
)

// EmptyHash ...
var EmptyHash [32]byte

// EmptyStepVotes ...
var EmptyStepVotes = message.StepVotes{}

// EmptyResult ...
var EmptyResult *Result

func init() {
	EmptyResult = &Result{
		EmptyHash[:],
		EmptyStepVotes,
	}
}

// Result of the Reduction steps
type Result struct {
	Hash []byte
	SV   message.StepVotes
}

// IsEmpty tests if the result of the aggregation is empty
func (r *Result) IsEmpty() bool {
	return r == EmptyResult
}

// Reduction is a struct to be embedded in the reduction steps
type Reduction struct {
	*consensus.Emitter
	TimeOut time.Duration
}

// IncreaseTimeout ...
func (r *Reduction) IncreaseTimeout(round uint64) {

	// if we converged on an empty block hash, we increase the timeout
	r.TimeOut = r.TimeOut * 2
	if r.TimeOut > 60*time.Second {
		lg.
			WithField("timeout", r.TimeOut).
			WithField("round", round).
			Error("max_timeout_reached")
		r.TimeOut = 60 * time.Second
	}
}

// SendReduction to the other peers
func (r *Reduction) SendReduction(round uint64, step uint8, hash []byte) error {
	hdr := header.Header{
		Round:     round,
		Step:      step,
		BlockHash: hash,
		PubKeyBLS: r.Keys.BLSPubKeyBytes,
	}

	sig, err := r.Sign(hdr)
	if err != nil {
		return err
	}

	red := message.NewReduction(hdr)
	red.SignedHash = sig
	_ = r.Gossip(message.New(topics.Reduction, *red))
	return nil
}

// ShouldProcess checks whether a message is consistent with the current round
// and step. If it is not, it either discards it or stores it for later. The
// function potentially mutates the consensus.Queue
func ShouldProcess(m message.Message, round uint64, step uint8, queue *consensus.Queue) bool {
	msg := m.Payload().(consensus.InternalPacket)
	hdr := msg.State()

	cmp := hdr.CompareRoundAndStep(round, step)
	if cmp == header.Before {
		lg.
			WithFields(log.Fields{
				"topic":             "Agreement",
				"round":             hdr.Round,
				"coordinator_round": round,
			}).
			Debugln("discarding obsolete agreement")
		return false
	}

	if cmp == header.After {
		lg.
			WithFields(log.Fields{
				"topic":             "Agreement",
				"round":             hdr.Round,
				"coordinator_round": round,
			}).
			Debugln("storing future round for later")
		queue.PutEvent(hdr.Round, hdr.Step, m)
		return false
	}

	if m.Category() != topics.Reduction {
		lg.
			WithFields(log.Fields{
				"topic":             "Reduction",
				"round":             hdr.Round,
				"coordinator_round": round,
			}).
			Debugln("message is not topics.Reduction")
		return false
	}

	return true
}
