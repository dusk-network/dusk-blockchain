package selection

import (
	"bytes"
	"errors"
	"math/big"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/zkproof"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type (
	ScoreHandler struct {
		bidList user.BidList
	}

	BidListCollector struct {
		BidListChan chan user.BidList
	}
)

func createSubcriber(eventBus *wire.EventBus) chan user.BidList {
	bidListChan := make(chan user.BidList)
	collector := &BidListCollector{bidListChan}
	wire.NewEventSubscriber(eventBus, collector, string(msg.BidListTopic)).Accept()
	return bidListChan
}

func (l *BidListCollector) Collect(r *bytes.Buffer) error {
	bidList, err := user.ReconstructBidListSubset(r.Bytes())
	if err != nil {
		return nil
	}
	l.BidListChan <- bidList
	return nil
}

func NewScoreHandler(eventBus *wire.EventBus) *ScoreHandler {
	bidListChan := createSubcriber(eventBus)
	sh := &ScoreHandler{}
	go func() {
		bidList := <-bidListChan
		sh.bidList = bidList
	}()
	return sh
}

func (p *ScoreHandler) NewEvent() wire.Event {
	return &ScoreEvent{}
}

func (p *ScoreHandler) Stage(e wire.Event) (uint64, uint8) {
	ev := e.(*ScoreEvent)
	return ev.Round, ev.Step
}

// Priority returns true if the
func (p *ScoreHandler) Priority(first, second wire.Event) wire.Event {
	ev1 := first.(*ScoreEvent)
	ev2 := second.(*ScoreEvent)
	score1 := big.NewInt(0).SetBytes(ev1.Score).Uint64()
	score2 := big.NewInt(0).SetBytes(ev2.Score).Uint64()
	if score1 < score2 {
		return ev2
	}

	return ev1
}

func (p *ScoreHandler) Verify(ev wire.Event) error {
	m := ev.(*ScoreEvent)

	// Check first if the BidList contains valid bids
	if err := p.validateBidListSubset(m.BidListSubset); err != nil {
		return err
	}

	// Verify the proof
	seedScalar := ristretto.Scalar{}
	seedScalar.Derive(m.Seed)
	if !zkproof.Verify(m.Proof, seedScalar.Bytes(), m.BidListSubset, m.Score, m.Z) {
		return errors.New("proof verification failed")
	}

	return nil
}

func (p *ScoreHandler) validateBidListSubset(bidListSubsetBytes []byte) error {
	bidListSubset, err := user.ReconstructBidListSubset(bidListSubsetBytes)
	if err != nil {
		return err
	}

	return p.bidList.ValidateBids(bidListSubset)
}
