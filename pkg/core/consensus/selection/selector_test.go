package selection

import (
	"bytes"
	"testing"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"

	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
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
		selector.RLock()
		defer selector.RUnlock()
		assert.Nil(t, selector.bestEvent)
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

func (m *mockScoreHandler) Priority(ev1, ev2 wire.Event) wire.Event {
	return ev2
}

func (m *mockScoreHandler) UpdateBidList(bL user.BidList) {}

func newMockHandler() consensus.EventHandler {
	mockEventHandler := &mocks.EventHandler{}
	mockEventHandler.On("Verify", mock.Anything).Return(nil)
	mockEventHandler.On("Marshal", mock.Anything, mock.Anything).Return(nil)
	return mockEventHandler
}

func newMockEvent() wire.Event {
	mockEvent := &mocks.Event{}
	mockEvent.On("Sender").Return([]byte{})
	mockEvent.On("Equal", mock.Anything).Return(true)
	return mockEvent
}
