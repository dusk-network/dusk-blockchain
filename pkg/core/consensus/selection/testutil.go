package selection

import (
	"bytes"
	"crypto/ed25519"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-wallet/key"
)

type mockSigner struct {
	keys key.ConsensusKeys
	bus  *eventbus.EventBus
}

func (m *mockSigner) Sign([]byte, []byte) ([]byte, error) {
	return make([]byte, 33), nil
}

func (m *mockSigner) SendAuthenticated(topic topics.Topic, blockHash []byte, payload *bytes.Buffer, id uint32) error {
	pubKeyBuf := new(bytes.Buffer)
	if err := encoding.WriteVarBytes(pubKeyBuf, m.keys.BLSPubKeyBytes); err != nil {
		return err
	}

	stateBuf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(stateBuf, 1); err != nil {
		return err
	}

	if err := encoding.WriteUint8(stateBuf, 1); err != nil {
		return err
	}

	buf, err := header.Compose(*pubKeyBuf, *stateBuf, blockHash)
	if err != nil {
		return err
	}

	if _, err := buf.ReadFrom(payload); err != nil {
		return err
	}

	signed, err := addEd25519Header(topic, m.keys, &buf)
	m.bus.Publish(topics.Gossip, signed)
	return err
}

func (m *mockSigner) SendWithHeader(topic topics.Topic, hash []byte, b *bytes.Buffer, id uint32) error {
	// Because the buffer in a BestScore message is empty, we will write the hash to it.
	// This way, we can check for correctness during tests.
	if err := encoding.Write256(b, hash); err != nil {
		return err
	}

	m.bus.Publish(topic, b)
	return nil
}

func addEd25519Header(topic topics.Topic, keys key.ConsensusKeys, buf *bytes.Buffer) (*bytes.Buffer, error) {
	edSigned := ed25519.Sign(*keys.EdSecretKey, buf.Bytes())

	// messages start from the signature
	whole := new(bytes.Buffer)
	if err := encoding.Write512(whole, edSigned); err != nil {
		return nil, err
	}

	// adding Ed public key
	if err := encoding.Write256(whole, keys.EdPubKeyBytes); err != nil {
		return nil, err
	}

	// adding marshalled header+payload
	if _, err := whole.ReadFrom(buf); err != nil {
		return nil, err
	}

	// prepending topic
	if err := topics.Prepend(whole, topic); err != nil {
		return nil, err
	}

	return whole, nil
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
	keys, _ := key.NewRandConsensusKeys()
	hlp := &Helper{
		Factory:       factory,
		BidList:       bidList,
		Selector:      sel,
		SimplePlayer:  consensus.NewSimplePlayer(),
		signer:        &mockSigner{keys, eb},
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
	evs := make([]consensus.Event, 0, len(h.BidList))
	for i := 0; i < len(h.BidList); i++ {
		ev := MockSelectionEventBuffer(hash)
		keys, _ := key.NewRandConsensusKeys()
		hdr := header.Header{
			Round:     h.Round,
			Step:      h.Step(),
			PubKeyBLS: keys.BLSPubKeyBytes,
			BlockHash: hash,
		}
		evs = append(evs, consensus.Event{hdr, *ev})
	}
	return evs
}

func (h *Helper) StartSelection() {
	h.Selector.startSelection()
}

// GenerateEd25519Fields will return the Ed25519 header fields for a given consensus
// event.
func (h *Helper) GenerateEd25519Fields(ev consensus.Event) []byte {
	payload := new(bytes.Buffer)
	if err := header.Marshal(payload, ev.Header); err != nil {
		panic(err)
	}

	if _, err := payload.ReadFrom(&ev.Payload); err != nil {
		panic(err)
	}

	keys, _ := key.NewRandConsensusKeys()
	buf, err := addEd25519Header(topics.Score, keys, payload)
	if err != nil {
		panic(err)
	}

	edFields := make([]byte, 96)
	if _, err := buf.Read(edFields); err != nil {
		panic(err)
	}

	return edFields
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
