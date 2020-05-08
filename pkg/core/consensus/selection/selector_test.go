package selection_test

import (
	"bytes"
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestSelection(t *testing.T) {
	bus := eventbus.New()
	hlp := selection.NewHelper(bus)
	// Sub to BestScore, to observe the outcome of the Selector
	bestScoreChan := make(chan message.Message, 1)
	bus.Subscribe(topics.BestScore, eventbus.NewChanListener(bestScoreChan))
	hlp.Initialize(consensus.MockRoundUpdate(1, nil, hlp.BidList))

	// Make sure to replace the handler, to avoid zkproof verification
	hlp.SetHandler(newMockHandler())

	hlp.StartSelection()

	// Send a set of events
	hash, _ := crypto.RandEntropy(32)
	hlp.SendBatch(hash)

	// Wait for a result on the best score channel
	bestScoreMsg := <-bestScoreChan
	score := bestScoreMsg.Payload().(message.Score)

	// We should've gotten a non-zero result
	assert.NotEqual(t, make([]byte, 32), score.Score)
}

func TestMultipleVerification(t *testing.T) {
	bus := eventbus.New()
	hlp := selection.NewHelper(bus)
	// Sub to gossip, so we can catch any outgoing events
	gossipChan := make(chan message.Message, 10)
	bus.Subscribe(topics.Gossip, eventbus.NewChanListener(gossipChan))
	hlp.Initialize(consensus.MockRoundUpdate(1, nil, hlp.BidList))

	// Make sure to replace the handler, to avoid zkproof verification
	hlp.SetHandler(newMockHandler())

	hlp.StartSelection()

	// Create some score messages
	hash, _ := crypto.RandEntropy(32)
	scores := hlp.Spawn(hash)

	// Sort the slice so that the highest scoring message is first
	sort.Slice(scores, func(i, j int) bool { return bytes.Compare(scores[i].Score, scores[j].Score) == 1 })

	// Now send them to the selector concurrently, except for the first one, with a
	// slight delay (guaranteeing that the highest scoring message comes in first).
	// Only one event should be verified and propagated.
	var wg sync.WaitGroup
	wg.Add(len(scores) - 1)
	go func() {
		time.Sleep(100 * time.Millisecond)
		for i, ev := range scores {
			if i != 0 {
				hlp.Selector.CollectScoreEvent(ev)
				wg.Done()
			}
		}
	}()

	hlp.Selector.CollectScoreEvent(scores[0])
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

// Mock implementation of a selection.Handler to avoid elaborate set-up of
// a Rust process for the purposes of zkproof verification.
type mockHandler struct {
}

func newMockHandler() *mockHandler {
	return &mockHandler{}
}

func (m *mockHandler) Verify(context.Context, uint64, uint8, message.Score) error {
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (m *mockHandler) LowerThreshold() {}
func (m *mockHandler) ResetThreshold() {}

func (m *mockHandler) Priority(first, second message.Score) bool {
	return bytes.Compare(second.Score, first.Score) != 1
}
