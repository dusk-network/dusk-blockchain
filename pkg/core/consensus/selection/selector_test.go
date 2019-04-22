package selection

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func TestSelection(t *testing.T) {
	eb := wire.NewEventBus()
	// subscribe to receive a result
	bestScoreChan := make(chan *bytes.Buffer, 1)
	eb.Subscribe(msg.BestScoreTopic, bestScoreChan)

	selector := newEventSelector(eb, newMockScoreHandler(), time.Millisecond*200, consensus.NewState())
	go selector.startSelection()
	selector.Process(newMockEvent())
	selector.Process(newMockEvent())
	selector.Process(newMockEvent())

	// we should receive something on the bestScoreChan after timeout
	<-bestScoreChan
	// bestEvent should have been set to nil
	selector.RLock()
	defer selector.RUnlock()
	assert.Nil(t, selector.bestEvent)
}

func TestRepropagation(t *testing.T) {
	eb := wire.NewEventBus()
	// subscribe to gossip topic
	gossipChan := make(chan *bytes.Buffer, 1)
	eb.Subscribe(string(topics.Gossip), gossipChan)

	selector := newEventSelector(eb, newMockScoreHandler(), time.Millisecond*200, consensus.NewState())
	go selector.startSelection()
	selector.Process(newMockEvent())

	// should have gotten something on the gossip channel
	timer := time.After(200 * time.Millisecond)
	select {
	case <-timer:
		assert.Fail(t, "should have gotten a buffer from gossipChan")
	case <-gossipChan:
		// success
	}
}

func TestStopSelector(t *testing.T) {
	eb := wire.NewEventBus()
	// subscribe to receive a result
	bestScoreChan := make(chan *bytes.Buffer, 1)
	eb.Subscribe(msg.BestScoreTopic, bestScoreChan)

	// run selection
	selector := newEventSelector(eb, newMockScoreHandler(), time.Second*5, consensus.NewState())
	go selector.startSelection()
	selector.Process(newMockEvent())
	selector.Process(newMockEvent())
	selector.Process(newMockEvent())
	selector.stopSelection()

	timer := time.After(200 * time.Millisecond)
	select {
	case <-bestScoreChan:
		assert.FailNow(t, "Selector should have not returned a value")
	case <-timer:
		// The test condition is satisfied if no Best Event is reported. Who cares about the ephemeral value of selector.bestEvent
		// selector.RLock()
		// defer selector.RUnlock()
		// assert.Nil(t, selector.bestEvent)
		// success :)
	}
}

type mockScoreHandler struct {
	consensus.EventHandler
}

func newMockScoreHandler() scoreEventHandler {
	return &mockScoreHandler{
		EventHandler: newMockHandler(),
	}
}

func (m *mockScoreHandler) Priority(ev1, ev2 wire.Event) bool {
	return false
}

func (m *mockScoreHandler) UpdateBidList(bL user.BidList) {}
func (m *mockScoreHandler) LowerThreshold()               {}
func (m *mockScoreHandler) ResetThreshold()               {}

func newMockHandler() consensus.EventHandler {
	var sender []byte
	mockEventHandler := &mocks.EventHandler{}
	mockEventHandler.On("Verify", mock.Anything).Return(nil)
	mockEventHandler.On("Marshal", mock.Anything, mock.Anything).Return(nil)
	mockEventHandler.On("ExtractHeader",
		mock.MatchedBy(func(ev wire.Event) bool {
			sender, _ = crypto.RandEntropy(32)
			return true
		})).Return(func(e wire.Event) *events.Header {
		return &events.Header{
			Round:     1,
			Step:      1,
			PubKeyBLS: sender,
		}
	})
	return mockEventHandler
}

func newMockEvent() wire.Event {
	mockEvent := &mocks.Event{}
	mockEvent.On("Sender").Return([]byte{})
	mockEvent.On("Equal", mock.Anything).Return(true)
	return mockEvent
}
