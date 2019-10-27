package selection_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestSelection(t *testing.T) {
	bus := eventbus.New()
	hlp := selection.NewHelper(bus, &mockPlayer{}, &mockSigner{bus})
	// Sub to BestScore, to observe the outcome of the Selector
	bestScoreChan := make(chan bytes.Buffer, 1)
	bus.Subscribe(topics.BestScore, eventbus.NewChanListener(bestScoreChan))

	// Start selection with a round update
	hlp.Initialize(consensus.MockRoundUpdate(1, nil, hlp.BidList))

	// Make sure to replace the handler, to avoid zkproof verification
	hlp.SetHandler(newMockHandler())

	// Send a set of events
	hash, _ := crypto.RandEntropy(32)
	hlp.SendBatch(hash)

	// Wait for a result on the best score channel
	evBuf := <-bestScoreChan
	h := make([]byte, 32)
	if err := encoding.Read256(&evBuf, h); err != nil {
		t.Fatal(err)
	}

	// We should've gotten a non-zero result
	assert.NotEqual(t, make([]byte, 32), h)
}

// No-op implementation of consensus.EventPlayer
type mockPlayer struct{}

func (m *mockPlayer) Resume(uint32) {}
func (m *mockPlayer) Pause(uint32)  {}
func (m *mockPlayer) Forward()      {}

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

// Mock implementation of a selection.Handler to avoid elaborate set-up of
// a Rust process for the purposes of zkproof verification.
type mockHandler struct {
}

func newMockHandler() *mockHandler {
	return &mockHandler{}
}

func (m *mockHandler) Verify(*selection.ScoreEvent) error {
	return nil
}

func (m *mockHandler) LowerThreshold() {}
func (m *mockHandler) ResetThreshold() {}

func (m *mockHandler) Priority(first, second *selection.ScoreEvent) bool {
	return bytes.Compare(second.Score, first.Score) != 1
}
