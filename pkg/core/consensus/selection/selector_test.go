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
	hlp := selection.NewHelper(bus)
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
	// Ensure `Forward()` was called
	assert.Equal(t, uint8(2), hlp.Step())
}

// Mock implementation of a selection.Handler to avoid elaborate set-up of
// a Rust process for the purposes of zkproof verification.
type mockHandler struct {
}

func newMockHandler() *mockHandler {
	return &mockHandler{}
}

func (m *mockHandler) Verify(*selection.Score) error {
	return nil
}

func (m *mockHandler) LowerThreshold() {}
func (m *mockHandler) ResetThreshold() {}

func (m *mockHandler) Priority(first, second *selection.Score) bool {
	return bytes.Compare(second.Score, first.Score) != 1
}
