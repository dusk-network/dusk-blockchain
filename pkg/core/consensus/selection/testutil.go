package selection

import (
	"bytes"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type mockSigner struct {
	bus *eventbus.EventBus
}

func (m *mockSigner) Sign([]byte, []byte) ([]byte, error) {
	return make([]byte, 33), nil
}

func (m *mockSigner) SendAuthenticated(topics.Topic, []byte, *bytes.Buffer) error { return nil }

func (m *mockSigner) SendWithHeader(topic topics.Topic, hash []byte, b *bytes.Buffer) error {
	// Because the buffer in a BestScore message is empty, we will write the hash to it.
	// This way, we can check for correctness during tests.
	if err := encoding.Write256(b, hash); err != nil {
		return err
	}

	m.bus.Publish(topic, b)
	return nil
}

// Helper for reducing selection test boilerplate
type Helper struct {
	*Factory
	BidList  user.BidList
	Selector *Selector
	*consensus.SimplePlayer
	signer consensus.Signer

	BestScoreChan chan bytes.Buffer
}

// NewHelper creates a Helper
func NewHelper(eb *eventbus.EventBus) *Helper {
	bidList := consensus.MockBidList(10)
	factory := NewFactory(eb, 1000*time.Millisecond)
	s := factory.Instantiate()
	sel := s.(*Selector)
	hlp := &Helper{
		Factory:       factory,
		BidList:       bidList,
		Selector:      sel,
		SimplePlayer:  consensus.NewSimplePlayer(),
		signer:        &mockSigner{eb},
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
	h.Selector.Initialize(h, h.signer, ru)
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
