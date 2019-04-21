package selection

import (
	"bytes"
	"errors"
	"sync"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"
	"gitlab.dusk.network/dusk-core/zkproof"
)

var (
	// Threshold number that a score needs to be greater than in order to be considered
	// for selection. Messages with scores lower than this threshold should not be
	// repropagated.
	threshold = []byte{170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170,
		170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170,
		170, 170, 170, 170, 170, 170}

	errScoreBelowThreshold     = errors.New("score below threshold")
	errProofFailedVerification = errors.New("proof verification failed")
)

type (
	scoreHandler struct {
		sync.RWMutex
		bidList user.BidList
	}

	scoreEventHandler interface {
		consensus.EventHandler
		wire.EventPrioritizer
		UpdateBidList(user.BidList)
	}
)

// NewScoreHandler returns a ScoreHandler, which encapsulates specific operations
// (e.g. verification, validation, marshalling and unmarshalling)
func newScoreHandler() *scoreHandler {
	return &scoreHandler{
		RWMutex: sync.RWMutex{},
	}
}

func (p *scoreHandler) NewEvent() wire.Event {
	return &ScoreEvent{}
}

func (p *scoreHandler) Unmarshal(r *bytes.Buffer, e wire.Event) error {
	return UnmarshalScoreEvent(r, e)
}

func (p *scoreHandler) Marshal(r *bytes.Buffer, e wire.Event) error {
	return MarshalScoreEvent(r, e)
}

func (p *scoreHandler) UpdateBidList(bidList user.BidList) {
	p.Lock()
	defer p.Unlock()
	p.bidList = bidList
}

func (p *scoreHandler) ExtractHeader(e wire.Event) *events.Header {
	ev := e.(*ScoreEvent)
	return &events.Header{
		Round: ev.Round,
	}
}

// Priority returns true if the first element has priority over the second, false otherwise
func (p *scoreHandler) Priority(first, second wire.Event) bool {
	ev1, ok := first.(*ScoreEvent)
	if !ok {
		// this happens when first is nil, in which case we should return second
		return false
	}

	ev2 := second.(*ScoreEvent)
	return bytes.Compare(ev2.Score, ev1.Score) != 1
}

func (p *scoreHandler) Verify(ev wire.Event) error {
	m := ev.(*ScoreEvent)

	// Check threshold
	if err := p.checkThreshold(m.Score); err != nil {
		return err
	}

	// Check if the BidList contains valid bids
	if err := p.validateBidListSubset(m.BidListSubset); err != nil {
		return err
	}

	// Verify the proof
	seedScalar := ristretto.Scalar{}
	seedScalar.Derive(m.Seed)

	proof := zkproof.ZkProof{
		Proof:         m.Proof,
		Score:         m.Score,
		Z:             m.Z,
		BinaryBidList: m.BidListSubset,
	}

	if !proof.Verify(seedScalar) {
		return errProofFailedVerification
	}

	return nil
}

func (p *scoreHandler) checkThreshold(score []byte) error {
	if bytes.Compare(score, threshold) == -1 {
		return errScoreBelowThreshold
	}
	return nil
}

func (p *scoreHandler) validateBidListSubset(bidListSubsetBytes []byte) *prerror.PrError {
	bidListSubset, err := user.ReconstructBidListSubset(bidListSubsetBytes)
	if err != nil {
		return err
	}

	p.Lock()
	defer p.Unlock()
	return p.bidList.ValidateBids(bidListSubset)
}
