// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package reduction

import (
	"context"
	"encoding/hex"
	"errors"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	log "github.com/sirupsen/logrus"
)

var (
	// ErrLowBlockHeight block height is lower or equal than blockchain tip height.
	ErrLowBlockHeight = errors.New("block height is low")
	errEmptyBlockHash = errors.New("empty hash")
)

// EmptyStepVotes ...
var EmptyStepVotes = message.StepVotes{}

// EmptyResult ...
var EmptyResult *Result

func init() {
	EmptyResult = &Result{
		block.EmptyHash[:],
		EmptyStepVotes,
	}
}

// Result of the Reduction steps.
type Result struct {
	Hash []byte
	SV   message.StepVotes
}

// IsEmpty tests if the result of the aggregation is empty.
func (r *Result) IsEmpty() bool {
	return r == EmptyResult
}

// Reduction is a struct to be embedded in the reduction steps.
type Reduction struct {
	*consensus.Emitter
	TimeOut time.Duration

	// VerifyFn verifies candidate block
	VerifyFn consensus.CandidateVerificationFunc
}

// IncreaseTimeout is used when reduction does not reach the quorum or
// converges over an empty block.
func (r *Reduction) IncreaseTimeout(round uint64) {
	// if we converged on an empty block hash, we increase the timeout
	r.TimeOut = r.TimeOut * 2
	if r.TimeOut > 60*time.Second {
		lg.
			WithField("timeout", r.TimeOut).
			WithField("round", round).
			Warn("max_timeout_reached")

		r.TimeOut = 60 * time.Second
	}
}

// verifyWithDelay calls verifyFn upon the candidate block but also incorporates a
// delay on success verification.
// vHash param is hash of block that has been already verified.
func (r *Reduction) verifyWithDelay(ctx context.Context, candidate *block.Block, step uint8) ([]byte, error) {
	if candidate == nil {
		return nil, errors.New("nil candidate")
	}

	if candidate.IsZero() {
		return nil, errEmptyBlockHash
	}

	hash, err := candidate.CalculateHash()
	if err != nil {
		return nil, err
	}

	st := time.Now().UnixMilli()

	if err := r.VerifyFn(ctx, *candidate); err != nil {
		return nil, err
	}

	// Candidate block is fully valid.
	// Vote for it.

	// Enable the delay if iteration is higher than 1
	if step > 3 {
		maxDelay := config.Get().Consensus.ThrottleIterMilli
		if maxDelay == 0 {
			maxDelay = 1000
		}

		d, err := util.Delay(st, maxDelay)
		if err == nil {
			log.WithField("step", step).WithField("sleep_for", d.String()).Trace("vst delayed")
		}
	}

	return hash, nil
}

// SendReduction propagates a signed vote for the candidate block, if block is
// fully valid. A full block validation will be performed if Candidate Hash differs from r.VerifiedHash.
// Error is returned only if the reduction should be not be registered locally.
func (r *Reduction) SendReduction(ctx context.Context, round uint64, step uint8, candidate *block.Block) (message.Message, []byte, error) {
	voteHash, err := r.verifyWithDelay(ctx, candidate, step)
	if err != nil {
		logVerifyErr(err, round, step, candidate)

		// On these errors the provisioner should not vote with emptyHash.
		switch err {
		case ErrLowBlockHeight, context.Canceled:
			return nil, nil, err
		}

		// Vote for an empty hash
		voteHash = block.EmptyHash[:]
	}

	// Generate Reduction message to propagate my vote.
	hdr := header.Header{
		Round:     round,
		Step:      step,
		BlockHash: voteHash,
		PubKeyBLS: r.Keys.BLSPubKey,
	}

	sig, err := r.Sign(hdr)
	if err != nil {
		panic(err)
	}

	red := message.NewReduction(hdr)
	red.SignedHash = sig

	m := message.New(topics.Reduction, *red)
	return m, voteHash, nil
}

// ShouldProcess checks whether a message is consistent with the current round
// and step. If it is not, it either discards it or stores it for later. The
// function potentially mutates the consensus.Queue.
func ShouldProcess(m message.Message, round uint64, step uint8, queue *consensus.Queue) bool {
	msg := m.Payload().(consensus.InternalPacket)
	hdr := msg.State()

	cmp := hdr.CompareRoundAndStep(round, step)
	if cmp == header.Before {
		lg.
			WithFields(log.Fields{
				"topic":          m.Category(),
				"round":          hdr.Round,
				"step":           hdr.Step,
				"expected round": round,
			}).
			Debugln("discarding obsolete event")
		return false
	}

	// Only store events up to 10 rounds into the future
	// XXX: According to protocol specs, we should abandon consensus
	// if we notice valid messages from far into the future.
	if cmp == header.After && hdr.Round-round < 10 {
		lg.
			WithFields(log.Fields{
				"topic":          m.Category(),
				"round":          hdr.Round,
				"step":           hdr.Step,
				"expected round": round,
			}).
			Debugln("storing future event for later")
		queue.PutEvent(hdr.Round, hdr.Step, m)
		return false
	}

	if m.Category() != topics.Reduction {
		lg.
			WithFields(log.Fields{
				"topic": m.Category(),
				"round": hdr.Round,
				"step":  hdr.Step,
			}).
			Warnln("unexpected topic for this step")
		return false
	}

	return true
}

func logVerifyErr(err error, round uint64, step uint8, candidate *block.Block) {
	l := log.
		WithError(err).
		WithField("round", round).
		WithField("step", step)

	if candidate != nil && candidate.Header != nil {
		l = l.WithField("candidate_hash", hex.EncodeToString(candidate.Header.Hash)).
			WithField("candidate_height", candidate.Header.Height)
	}

	// errEmptyBlockHash could be returned here if either Selection or
	// 1st_reduction steps experience a timeout event

	// ErrLowBlockHeight could be returned if consensus state is behind the chain state.

	switch err {
	case errEmptyBlockHash, ErrLowBlockHeight:
		l.Debug("verifyfn failed")
	default:
		// a problem to report as an error to investigate
		l.WithError(err).Error("verifyfn failed")
	}
}
