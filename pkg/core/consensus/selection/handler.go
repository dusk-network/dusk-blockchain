package selection

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
)

var _ Handler = (*ScoreHandler)(nil)

type (
	// ScoreHandler manages the score threshold, performs verification of
	// message.Score, keeps tab of the highest score so far
	ScoreHandler struct {
		bidList user.BidList

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
		Verify(context.Context, message.Score) error
		ResetThreshold()
		LowerThreshold()
		Priority(message.Score, message.Score) bool
	}
)

// NewScoreHandler returns a new instance if ScoreHandler
func NewScoreHandler(bidList user.BidList, scoreVerifier transactions.Provisioner) *ScoreHandler {
	return &ScoreHandler{
		bidList:       bidList,
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
func (sh *ScoreHandler) Verify(ctx context.Context, m message.Score) error {
	// Check threshold
	sh.lock.RLock()
	defer sh.lock.RUnlock()
	if sh.threshold.Exceeds(m.Score) {
		return errors.New("threshold exceeds score")
	}

	// Check if the BidList contains valid bids
	if err := sh.validateBidListSubset(m.BidListSubset); err != nil {
		return err
	}

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(500*time.Millisecond))
	defer cancel()

	return sh.scoreVerifier.VerifyScore(ctx, transactions.Score{
		Proof: m.Proof,
		Score: m.Score,
		Z:     m.Z,
		Bids:  m.BidListSubset,
	})

	// Verify the proof
	//seedScalar := ristretto.Scalar{}
	//seedScalar.Derive(m.Seed)

	//proof := zkproof.ZkProof{
	//	Proof:         m.Proof,
	//	Score:         m.Score,
	//	Z:             m.Z,
	//	BinaryBidList: m.BidListSubset,
	//}

	//if !proof.Verify(seedScalar) {
	//	return errors.New("proof verification failed")
	//}
}

func (sh *ScoreHandler) validateBidListSubset(bidListSubsetBytes []byte) error {
	bidListSubset, err := user.ReconstructBidListSubset(bidListSubsetBytes)
	if err != nil {
		return err
	}

	return sh.bidList.ValidateBids(bidListSubset)
}
