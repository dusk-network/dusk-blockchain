package consensus_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

var empty struct{}

func TestRelevantEvent(t *testing.T) {
	round := uint64(1)
	step := uint8(1)
	processChan := make(chan error, 1)
	eventFilter := newEventFilter(round, step, true, newMockEventProcessor(nil, processChan))
	eventFilter.UpdateRound(1)

	// Run collect with an empty buffer, as the event will be mocked
	assert.Nil(t, eventFilter.Collect(new(bytes.Buffer)))

	// We should get something from the processChan
	result := <-processChan
	// Result should be nil
	assert.Nil(t, result)
}

func TestEarlyEvent(t *testing.T) {
	round := uint64(2)
	step := uint8(1)
	processChan := make(chan error, 1)
	eventFilter := newEventFilter(round, step, true, newMockEventProcessor(nil, processChan))
	eventFilter.UpdateRound(1)

	// Run collect with an empty buffer, as the event will be mocked
	assert.Nil(t, eventFilter.Collect(new(bytes.Buffer)))
	// Queue should now hold an event
	// Update the round, and flush the queue to get it
	eventFilter.UpdateRound(2)
	eventFilter.FlushQueue()
	// We should get something from the processChan
	result := <-processChan
	// Result should be nil
	assert.Nil(t, result)
}

func TestObsoleteEvent(t *testing.T) {
	round := uint64(1)
	step := uint8(1)
	processChan := make(chan error, 1)
	eventFilter := newEventFilter(round, step, true, newMockEventProcessor(nil, processChan))
	eventFilter.UpdateRound(2)

	// Run collect with an empty buffer, as the event will be mocked
	assert.Nil(t, eventFilter.Collect(new(bytes.Buffer)))

	// We should not get anything from the processChan
	timer := time.After(100 * time.Millisecond)
	select {
	case <-processChan:
		t.Fatal("processor received an obsolete event")
	case <-timer:
	}
}

// newEventFilter simplifies the creation of an EventFilter with specific mocked
// components.
func newEventFilter(round uint64, step uint8, isMember bool,
	processor consensus.EventProcessor) *consensus.EventFilter {
	return consensus.NewEventFilter(newMockHandlerFilter(round, step),
		consensus.NewState(), processor, true)
}

func newMockEvent() wire.Event {
	mockEvent := &mocks.Event{}
	mockEvent.On("Sender").Return([]byte{})
	mockEvent.On("Equal", mock.Anything).Return(false)
	return mockEvent
}

func newMockHandlerFilter(round uint64, step uint8) consensus.EventHandler {
	mockEventHandler := &mocks.EventHandler{}
	mockEventHandler.On("NewEvent").Return(newMockEvent())
	mockEventHandler.On("Unmarshal", mock.Anything, mock.Anything).Return(nil)
	mockEventHandler.On("ExtractHeader", mock.Anything).Return(func(e wire.Event) *events.Header {
		return &events.Header{
			Round: round,
			Step:  step,
		}
	})
	return mockEventHandler
}

type mockEventProcessor struct {
	verifyErr   error
	processChan chan error
}

func newMockEventProcessor(verifyErr error, processChan chan error) consensus.EventProcessor {
	return &mockEventProcessor{
		verifyErr:   verifyErr,
		processChan: processChan,
	}
}

func (m *mockEventProcessor) Process(ev wire.Event) {
	m.processChan <- m.verifyErr
}
