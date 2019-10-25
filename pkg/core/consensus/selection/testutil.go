package selection

import (
	"bytes"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
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

	Handler *Handler
}

// NewHelper creates a Helper
func NewHelper(eb *eventbus.EventBus, eventPlayer consensus.EventPlayer, signer consensus.Signer) *Helper {
	bidList := consensus.MockBidList()
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
		Handler:       NewHandler(bidList),
	}
	hlp.createResultChan()
	return hlp
}

func (h *Helper) createResultChan() {
	listener := eventbus.NewChanListener(h.BestScoreChan)
	h.Bus.Subscribe(topics.BestScore, listener)
}

func (h *Helper) Initialize(ru consensus.RoundUpdate) {
	h.Selector.Initialize(h.eventPlayer, h.signer, ru)
}
