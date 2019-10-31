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
	hlp.Initialize(consensus.MockRoundUpdate(1, nil, hlp.BidList))

	// Make sure to replace the handler, to avoid zkproof verification
	hlp.SetHandler(newMockHandler())

	hlp.StartSelection()

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

// Ensure that the Ed25519 header of the score message is changed when repropagated
func TestSwapHeader(t *testing.T) {
	bus := eventbus.New()
	hlp := selection.NewHelper(bus)
	// Sub to gossip, so we can catch any outgoing events
	gossipChan := make(chan bytes.Buffer, 1)
	bus.Subscribe(topics.Gossip, eventbus.NewChanListener(gossipChan))
	hlp.Initialize(consensus.MockRoundUpdate(1, nil, hlp.BidList))

	// Make sure to replace the handler, to avoid zkproof verification
	hlp.SetHandler(newMockHandler())

	hlp.StartSelection()

	// Create some score messages
	hash, _ := crypto.RandEntropy(32)
	evs := hlp.Spawn(hash)

	// We save the Ed25519 fields for comparison later
	edFields := hlp.GenerateEd25519Fields(evs[0])

	// Send this event to the selector, and get it repropagated
	hlp.Selector.CollectScoreEvent(evs[0])

	// Catch the repropagated event
	repropagated := <-gossipChan
	repropagatedEdFields := make([]byte, 96)
	if _, err := repropagated.Read(repropagatedEdFields); err != nil {
		t.Fatal(err)
	}

	assert.NotEqual(t, edFields, repropagatedEdFields)
}

// Mock implementation of a selection.Handler to avoid elaborate set-up of
// a Rust process for the purposes of zkproof verification.
type mockHandler struct {
}

func newMockHandler() *mockHandler {
	return &mockHandler{}
}

func (m *mockHandler) Verify(selection.Score) error {
	return nil
}

func (m *mockHandler) LowerThreshold() {}
func (m *mockHandler) ResetThreshold() {}

func (m *mockHandler) Priority(first, second selection.Score) bool {
	return bytes.Compare(second.Score, first.Score) != 1
}
