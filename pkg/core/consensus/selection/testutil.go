package selection

import (
	"context"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// NOTE: looks like this is not used. In case, we need to use it, we need to
// add the Header from the Helper through a callback
type mockSigner struct {
	pubkey []byte
	bus    *eventbus.EventBus
}

func (m *mockSigner) Sign(header.Header) ([]byte, error) {
	return make([]byte, 33), nil
}

func (m *mockSigner) Gossip(msg message.Message, id uint32) error {
	// message.Marshal takes care of prepending the topic, marshaling the
	// header, etc
	buf, err := message.Marshal(msg)
	if err != nil {
		return err
	}

	serialized := message.New(msg.Category(), buf)

	// gossip away
	m.bus.Publish(topics.Gossip, serialized)
	return nil
}

func (m *mockSigner) Compose(pf consensus.PacketFactory) consensus.InternalPacket {
	return pf.Create(m.pubkey, 0, 1)
}

func (m *mockSigner) SendInternally(topic topics.Topic, msg message.Message, id uint32) error {
	m.bus.Publish(topic, msg)
	return nil
}

// Helper for reducing selection test boilerplate
type Helper struct {
	*Factory
	BidList  user.BidList
	Selector *Selector
	*consensus.SimplePlayer
	signer consensus.Signer

	BestScoreChan chan message.Message
}

// NewHelper creates a Helper
func NewHelper(eb *eventbus.EventBus) *Helper {
	bidList := consensus.MockBidList(10)
	mockProxy := transactions.MockProxy{
		P: transactions.PermissiveProvisioner{},
	}
	factory := NewFactory(context.Background(), eb, 1000*time.Millisecond, mockProxy)
	s := factory.Instantiate()
	sel := s.(*Selector)
	keys, _ := key.NewRandKeys()
	hlp := &Helper{
		Factory:       factory,
		BidList:       bidList,
		Selector:      sel,
		SimplePlayer:  consensus.NewSimplePlayer(),
		signer:        &mockSigner{keys.BLSPubKeyBytes, eb},
		BestScoreChan: make(chan message.Message, 1),
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
func (h *Helper) Spawn(hash []byte) []message.Score {
	evs := make([]message.Score, 0, len(h.BidList))
	for i := 0; i < len(h.BidList); i++ {
		keys, _ := key.NewRandKeys()
		hdr := header.Header{
			Round:     h.Round,
			Step:      h.Step(),
			PubKeyBLS: keys.BLSPubKeyBytes,
			BlockHash: hash,
		}
		evs = append(evs, message.MockScore(hdr, hash))
	}
	return evs
}

// StartSelection forces the Selector to start the selection
func (h *Helper) StartSelection() {
	h.Selector.startSelection()
}

// SendBatch generates a batch of score events and sends them to the selector.
func (h *Helper) SendBatch(hash []byte) {
	batch := h.Spawn(hash)
	var wg sync.WaitGroup
	// Tell the 'wg' WaitGroup how many threads/goroutines
	//   that are about to run concurrently.
	wg.Add(len(batch))
	for i := 0; i < len(batch); i++ {
		go func(i int) {
			defer wg.Done()
			ev := batch[i]
			_ = h.Selector.CollectScoreEvent(ev)
		}(i)
	}
}

// SetHandler sets the handler on the Selector. Used for bypassing zkproof
// verification calls during tests.
func (h *Helper) SetHandler(handler Handler) {
	h.Selector.handler = handler
}
