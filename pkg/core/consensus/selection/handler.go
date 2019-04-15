package selection

import (
	"bytes"
	"errors"
	"math/big"
	"sync"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"
	"gitlab.dusk.network/dusk-core/zkproof"
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

// NewScoreHandler returns a ScoreHandler, which encapsulates specific operations (e.g. verification, validation, marshalling and unmarshalling)
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

// Priority returns true if the
func (p *scoreHandler) Priority(first, second wire.Event) wire.Event {
	ev1, ok := first.(*ScoreEvent)
	if !ok {
		// this happens when first is nil, in which case we should return second
		return second
	}

	ev2 := second.(*ScoreEvent)
	score1 := big.NewInt(0).SetBytes(ev1.Score).Uint64()
	score2 := big.NewInt(0).SetBytes(ev2.Score).Uint64()
	if score1 < score2 {
		return ev2
	}

	return ev1
}

func (p *scoreHandler) Verify(ev wire.Event) error {
	m := ev.(*ScoreEvent)

	// Check first if the BidList contains valid bids
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
		return errors.New("proof verification failed")
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
