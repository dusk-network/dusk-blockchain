package consensus_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// Test that a relevant event gets passed down to the Processor properly.
func TestRelevantEvent(t *testing.T) {
	round := uint64(1)
	step := uint8(1)
	eventFilter := newEventFilter(round, step, true, true)
	eventFilter.UpdateRound(1)

	// Run collect with an empty buffer, as the event will be mocked
	assert.NoError(t, eventFilter.Collect(new(bytes.Buffer)))
	time.Sleep(200 * time.Millisecond)

	// We should get an event if we call accumulator.All
	evs := eventFilter.Accumulator.All()
	// Len should be 1
	assert.Equal(t, 1, len(evs))
}

// Test that early events get queued, and then passed down to the Processor properly
// upon the needed state change.
func TestEarlyEvent(t *testing.T) {
	round := uint64(2)
	step := uint8(1)
	eventFilter := newEventFilter(round, step, true, true)
	eventFilter.UpdateRound(1)

	assert.NoError(t, eventFilter.Collect(new(bytes.Buffer)))
	time.Sleep(200 * time.Millisecond)
	// Queue should now hold an event
	// Update the round, and flush the queue to get it
	eventFilter.UpdateRound(2)
	eventFilter.FlushQueue()
	time.Sleep(200 * time.Millisecond)
	// We should get an event if we call accumulator.All
	evs := eventFilter.Accumulator.All()
	// Len should be 1
	assert.Equal(t, 1, len(evs))
}

// Test that obsolete events get dropped.
func TestObsoleteEvent(t *testing.T) {
	round := uint64(1)
	step := uint8(1)
	eventFilter := newEventFilter(round, step, true, true)
	eventFilter.UpdateRound(2)

	assert.Nil(t, eventFilter.Collect(new(bytes.Buffer)))
	time.Sleep(200 * time.Millisecond)

	// We should not get anything from the processChan
	timer := time.After(100 * time.Millisecond)
	<-timer
	// We should get no events if we call accumulator.All
	evs := eventFilter.Accumulator.All()
	// Len should be 0
	assert.Equal(t, 0, len(evs))
}

// Test that the entire queue for a round is flushed and processed, when checkStep
// is set to false.
func TestFlushQueueNoCheckStep(t *testing.T) {
	round := uint64(2)
	step := uint8(2)
	eventFilter := newEventFilter(round, step, true, false)
	eventFilter.UpdateRound(1)

	assert.Nil(t, eventFilter.Collect(new(bytes.Buffer)))
	time.Sleep(200 * time.Millisecond)
	// Update round, and flush queue to get the event
	eventFilter.UpdateRound(2)
	eventFilter.FlushQueue()
	time.Sleep(200 * time.Millisecond)

	// We should get an event if we call accumulator.All
	evs := eventFilter.Accumulator.All()
	// Len should be 1
	assert.Equal(t, 1, len(evs))
}

// newEventFilter simplifies the creation of an EventFilter with specific mocked
// components.
func newEventFilter(round uint64, step uint8, isMember bool, checkStep bool) *consensus.EventFilter {
	return consensus.NewEventFilter(newMockHandlerAccumulator(round, step, nil, []byte{}, 2, "1", isMember),
		consensus.NewState(), checkStep)
}

func newMockEvent() wire.Event {
	mockEvent := &mocks.Event{}
	mockEvent.On("Sender").Return([]byte{})
	mockEvent.On("Equal", mock.Anything).Return(false)
	return mockEvent
}
