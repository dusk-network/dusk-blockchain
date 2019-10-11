package selection

import (
	"bytes"
	"errors"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/prerror"
	zkproof "github.com/dusk-network/dusk-zkproof"
)

type (
	scoreHandler struct {
		bidList user.BidList

		// Threshold number that a score needs to be greater than in order to be considered
		// for selection. Messages with scores lower than this threshold should not be
		// repropagated.
		threshold *consensus.Threshold
	}

	// ScoreEventHandler extends the consensus.EventHandler interface with methods
	// specific to the handling of score events.
	ScoreEventHandler interface {
		consensus.EventHandler
		wire.EventPrioritizer
		ResetThreshold()
		LowerThreshold()
	}
)

// NewScoreHandler returns a ScoreHandler, which encapsulates specific operations
// (e.g. verification, validation, marshalling and unmarshalling)
func newScoreHandler(bidList user.BidList) *scoreHandler {
	return &scoreHandler{
		bidList:   bidList,
		threshold: consensus.NewThreshold(),
	}
}

func (sh *scoreHandler) Deserialize(r *bytes.Buffer) (wire.Event, error) {
	ev := &ScoreEvent{}
	if err := sh.Unmarshal(r, ev); err != nil {
		return nil, err
	}
	return ev, nil
}

func (sh *scoreHandler) Unmarshal(r *bytes.Buffer, e wire.Event) error {
	return UnmarshalScoreEvent(r, e)
}

func (sh *scoreHandler) Marshal(r *bytes.Buffer, e wire.Event) error {
	return MarshalScoreEvent(r, e)
}

func (sh *scoreHandler) ExtractHeader(e wire.Event) *header.Header {
	ev := e.(*ScoreEvent)
	return &header.Header{
		Round: ev.Round,
	}
}

func (sh *scoreHandler) ResetThreshold() {
	sh.threshold.Reset()
}

func (sh *scoreHandler) LowerThreshold() {
	sh.threshold.Lower()
}

// Priority returns true if the first element has priority over the second, false otherwise
func (sh *scoreHandler) Priority(first, second wire.Event) bool {
	ev1, ok := first.(*ScoreEvent)
	if !ok {
		// this happens when first is nil, in which case we should return second
		return false
	}

	ev2 := second.(*ScoreEvent)
	return bytes.Compare(ev2.Score, ev1.Score) != 1
}

func (sh *scoreHandler) Verify(ev wire.Event) error {
	m := ev.(*ScoreEvent)

	// Check threshold
	if !sh.threshold.Exceeds(m.Score) {
		return errors.New("score does not exceed threshold")
	}

	// Check if the BidList contains valid bids
	if err := sh.validateBidListSubset(m.BidListSubset); err != nil {
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
		return errors.New("proof verification failed")
	}

	return nil
}

func (sh *scoreHandler) validateBidListSubset(bidListSubsetBytes []byte) *prerror.PrError {
	bidListSubset, err := user.ReconstructBidListSubset(bidListSubsetBytes)
	if err != nil {
		return err
	}

	return sh.bidList.ValidateBids(bidListSubset)
}
