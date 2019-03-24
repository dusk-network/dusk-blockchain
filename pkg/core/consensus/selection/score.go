package selection

import (
	"bytes"
	"errors"
	"math/big"
	"time"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/zkproof"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// LaunchScoreComponent creates and launches the component which responsibility is to validate and select the best score among the blind bidders
func LaunchScoreComponent(eventBus *wire.EventBus, timeout time.Duration) *broker {
	handler := newScoreHandler(eventBus)
	broker := newBroker(eventBus, handler, timeout, string(msg.BestScoreTopic))
	go broker.Listen()
	return broker
}

// InitBestScoreUpdate is the utility function to create and wire a channel for notifications of the best ScoreEvent
func InitBestScoreUpdate(eventBus *wire.EventBus) chan *ScoreEvent {
	bestScoreChan := make(chan *ScoreEvent, 1)
	collector := &scoreCollector{
		bestScoreChan: bestScoreChan,
		unMarshaller:  newScoreUnMarshaller(),
	}
	go wire.NewEventSubscriber(eventBus, collector, string(msg.BestScoreTopic)).Accept()
	return bestScoreChan
}

// InitBidListUpdate creates and initiates a channel for the updates in the BidList
func InitBidListUpdate(eventBus *wire.EventBus) chan user.BidList {
	bidListChan := make(chan user.BidList)
	collector := &bidListCollector{bidListChan}
	wire.NewEventSubscriber(eventBus, collector, string(msg.BidListTopic)).Accept()
	return bidListChan
}

type (
	scoreHandler struct {
		bidList      user.BidList
		unMarshaller *scoreUnMarshaller
	}

	bidListCollector struct {
		BidListChan chan user.BidList
	}

	//scoreCollector is a helper to obtain a score channel already wired to the EventBus and fully functioning
	scoreCollector struct {
		bestScoreChan chan *ScoreEvent
		unMarshaller  *scoreUnMarshaller
	}
)

func (sc *scoreCollector) Collect(r *bytes.Buffer) error {
	ev := &ScoreEvent{}
	if err := sc.unMarshaller.Unmarshal(r, ev); err != nil {
		return err
	}
	sc.bestScoreChan <- ev
	return nil
}

// Collect as defined in the EventCollector interface. It reconstructs the bidList and notifies about it
func (l *bidListCollector) Collect(r *bytes.Buffer) error {
	bidList, err := user.ReconstructBidListSubset(r.Bytes())
	if err != nil {
		return nil
	}
	l.BidListChan <- bidList
	return nil
}

// NewScoreHandler returns a ScoreHandler, which encapsulates specific operations (e.g. verification, validation, marshalling and unmarshalling)
func newScoreHandler(eventBus *wire.EventBus) *scoreHandler {
	bidListChan := InitBidListUpdate(eventBus)
	sh := &scoreHandler{
		unMarshaller: newScoreUnMarshaller(),
	}
	go func() {
		bidList := <-bidListChan
		sh.bidList = bidList
	}()
	return sh
}

func (p *scoreHandler) NewEvent() wire.Event {
	return &ScoreEvent{}
}

func (p *scoreHandler) Unmarshal(r *bytes.Buffer, e wire.Event) error {
	return p.unMarshaller.Unmarshal(r, e)
}

func (p *scoreHandler) Marshal(r *bytes.Buffer, e wire.Event) error {
	return p.unMarshaller.Marshal(r, e)
}

func (p *scoreHandler) ExtractHeader(e wire.Event, h *consensus.EventHeader) {
	ev := e.(*ScoreEvent)
	h.Round = ev.Round
	h.Step = ev.Step
	h.PubKeyBLS = ev.Z
}

// Priority returns true if the
func (p *scoreHandler) Priority(first, second wire.Event) wire.Event {
	ev1 := first.(*ScoreEvent)
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
	if !zkproof.Verify(m.Proof, seedScalar.Bytes(), m.BidListSubset, m.Score, m.Z) {
		return errors.New("proof verification failed")
	}

	return nil
}

func (p *scoreHandler) validateBidListSubset(bidListSubsetBytes []byte) error {
	bidListSubset, err := user.ReconstructBidListSubset(bidListSubsetBytes)
	if err != nil {
		return err
	}

	return p.bidList.ValidateBids(bidListSubset)
}
