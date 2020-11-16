package selection

import (
	"bytes"
	"context"
	"errors"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/blindbid"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/common"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
)

var _ Handler = (*ScoreHandler)(nil)

type (
	// ScoreHandler manages the score threshold, performs verification of
	// message.Score, keeps tab of the highest score so far
	ScoreHandler struct {
		// Threshold number that a score needs to be greater than in order to be considered
		// for selection. Messages with scores lower than this threshold should not be
		// repropagated.
		lock      sync.RWMutex
		threshold *consensus.Threshold

		scoreVerifier transactions.Provisioner
	}

	// Handler is an abstraction of the selection component event handler.
	// It is primarily used for testing purposes, to bypass the zkproof verification.
	Handler interface {
		Verify(context.Context, uint64, uint8, message.Score) error
		ResetThreshold()
		LowerThreshold()
		Priority(message.Score, message.Score) bool
	}
)

// NewScoreHandler returns a new instance if ScoreHandler
func NewScoreHandler(scoreVerifier transactions.Provisioner) *ScoreHandler {
	return &ScoreHandler{
		threshold:     consensus.NewThreshold(),
		scoreVerifier: scoreVerifier,
	}
}

// ResetThreshold resets the score threshold that sets the absolute minimum for
// a score to be eligible for sending
func (sh *ScoreHandler) ResetThreshold() {
	sh.lock.Lock()
	defer sh.lock.Unlock()
	sh.threshold.Reset()
}

// LowerThreshold lowers the threshold after a timespan when no BlockGenerator
// could send a valid score
func (sh *ScoreHandler) LowerThreshold() {
	sh.lock.Lock()
	defer sh.lock.Unlock()
	sh.threshold.Lower()
}

// Priority returns true if the first element has priority over the second, false otherwise
func (sh *ScoreHandler) Priority(first, second message.Score) bool {
	return bytes.Compare(second.Score, first.Score) != 1
}

// Verify a score by delegating the ZK library to validate the proof
func (sh *ScoreHandler) Verify(ctx context.Context, round uint64, step uint8, m message.Score) error {
	// Check threshold
	sh.lock.RLock()
	defer sh.lock.RUnlock()
	score := &common.BlsScalar{Data: m.Score}
	if sh.threshold.Exceeds(score) {
		return errors.New("threshold exceeds score")
	}

	return sh.scoreVerifier.VerifyScore(ctx, round, step, blindbid.VerifyScoreRequest{
		Proof:    &common.Proof{Data: m.Proof},
		Score:    score,
		Seed:     &common.BlsScalar{Data: m.Seed},
		ProverID: &common.BlsScalar{Data: m.Identity},
		Round:    round,
		Step:     uint32(step),
	})
}
