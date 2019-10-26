package selection

import (
	"bytes"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// Helper for reducing selection test boilerplate
type Helper struct {
	*Factory
	BidList     user.BidList
	Selector    *Selector
	eventPlayer consensus.EventPlayer
	signer      consensus.Signer

	BestScoreChan chan bytes.Buffer
}

// NewHelper creates a Helper
func NewHelper(eb *eventbus.EventBus, eventPlayer consensus.EventPlayer, signer consensus.Signer) *Helper {
	bidList := consensus.MockBidList(10)
	factory := NewFactory(eb, 1000*time.Millisecond)
	s := factory.Instantiate()
	sel := s.(*Selector)
	hlp := &Helper{
		Factory:       factory,
		BidList:       bidList,
		Selector:      sel,
		eventPlayer:   eventPlayer,
		signer:        signer,
		BestScoreChan: make(chan bytes.Buffer, 1),
	}
	hlp.createResultChan()
	return hlp
}

func (h *Helper) createResultChan() {
	listener := eventbus.NewChanListener(h.BestScoreChan)
	h.Bus.Subscribe(topics.BestScore, listener)
}

// Initialize the selector with the given round update.
func (h *Helper) Initialize(ru consensus.RoundUpdate) {
	h.Selector.Initialize(h.eventPlayer, h.signer, ru)
}

// Spawn a set of score events.
func (h *Helper) Spawn(hash []byte) []consensus.Event {
	evs := make([]consensus.Event, len(h.BidList))
	for i := 0; i < len(h.BidList); i++ {
		ev := MockSelectionEventBuffer(hash, h.BidList)
		// The header can remain empty, as we are not utilizing the Coordinator
		// for testing
		evs = append(evs, consensus.Event{header.Header{}, *ev})
	}
	return evs
}

// SendBatch generates a batch of score events and sends them to the selector.
func (h *Helper) SendBatch(hash []byte) {
	batch := h.Spawn(hash)
	for _, ev := range batch {
		go h.Selector.CollectScoreEvent(ev)
	}
}

// SetHandler sets the handler on the Selector. Used for bypassing zkproof
// verification calls during tests.
func (h *Helper) SetHandler(handler Handler) {
	h.Selector.handler = handler
}
