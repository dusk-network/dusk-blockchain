package selection_test

import (
	"bytes"
	"sort"
	"sync"
	"testing"
	"time"

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

func TestMultipleVerification(t *testing.T) {
	bus := eventbus.New()
	hlp := selection.NewHelper(bus)
	// Sub to gossip, so we can catch any outgoing events
	gossipChan := make(chan bytes.Buffer, 10)
	bus.Subscribe(topics.Gossip, eventbus.NewChanListener(gossipChan))
	hlp.Initialize(consensus.MockRoundUpdate(1, nil, hlp.BidList))

	// Make sure to replace the handler, to avoid zkproof verification
	hlp.SetHandler(newMockHandler())

	hlp.StartSelection()

	// Create some score messages
	hash, _ := crypto.RandEntropy(32)
	evs := hlp.Spawn(hash)

	// Sort the slice so that the highest scoring message is first
	sorted := sortEventsByScore(t, evs)

	// Now send them to the selector concurrently, except for the first one, with a
	// slight delay (guaranteeing that the highest scoring message comes in first).
	// Only one event should be verified and propagated.
	var wg sync.WaitGroup
	wg.Add(len(sorted) - 1)
	go func() {
		time.Sleep(100 * time.Millisecond)
		for i, ev := range sorted {
			if i != 0 {
				hlp.Selector.CollectScoreEvent(ev)
				wg.Done()
			}
		}
	}()

	hlp.Selector.CollectScoreEvent(sorted[0])
	wg.Wait()

	// We should only have one repropagated event
	assert.Equal(t, 1, len(gossipChan))
}

// Ensure that `CollectScoreEvent` does not panic if it is called before
// receiving a Generation message. (Note that this can happen, as the Score
// listener is not paused on startup)
func TestCollectNoGeneration(t *testing.T) {
	assert.NotPanics(t, func() {
		bus := eventbus.New()
		hlp := selection.NewHelper(bus)
		hlp.Initialize(consensus.MockRoundUpdate(1, nil, hlp.BidList))

		// Make sure to replace the handler, to avoid zkproof verification
		hlp.SetHandler(newMockHandler())

		// Create some score messages
		hash, _ := crypto.RandEntropy(32)
		evs := hlp.Spawn(hash)

		hlp.Selector.CollectScoreEvent(evs[0])
	})
}

func sortEventsByScore(t *testing.T, evs []consensus.Event) []consensus.Event {
	// Retrieve message from payload
	scoreEvs := make([]selection.Score, 0, len(evs))
	for _, ev := range evs {
		score := selection.Score{}
		if err := selection.UnmarshalScore(&ev.Payload, &score); err != nil {
			t.Fatal(err)
		}

		scoreEvs = append(scoreEvs, score)
	}

	// Sort by score, from highest to lowest
	sort.Slice(scoreEvs, func(i, j int) bool { return bytes.Compare(scoreEvs[i].Score, scoreEvs[j].Score) == 1 })

	// Set payload back on the messages
	sortedEvs := make([]consensus.Event, 0, len(evs))
	for i, ev := range evs {
		buf := new(bytes.Buffer)
		if err := selection.MarshalScore(buf, &scoreEvs[i]); err != nil {
			t.Fatal(err)
		}

		ev.Payload = *buf
		sortedEvs = append(sortedEvs, ev)
	}

	return sortedEvs
}

// Mock implementation of a selection.Handler to avoid elaborate set-up of
// a Rust process for the purposes of zkproof verification.
type mockHandler struct {
}

func newMockHandler() *mockHandler {
	return &mockHandler{}
}

func (m *mockHandler) Verify(selection.Score) error {
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (m *mockHandler) LowerThreshold() {}
func (m *mockHandler) ResetThreshold() {}

func (m *mockHandler) Priority(first, second selection.Score) bool {
	return bytes.Compare(second.Score, first.Score) != 1
}
