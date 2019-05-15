package consensus_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/header"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func TestRelevantEvent(t *testing.T) {
	round := uint64(1)
	step := uint8(1)
	processChan := make(chan error, 1)
	eventFilter := newEventFilter(round, step, true,
		newMockEventProcessor(nil, processChan), true)
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
	eventFilter := newEventFilter(round, step, true,
		newMockEventProcessor(nil, processChan), true)
	eventFilter.UpdateRound(1)

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
	eventFilter := newEventFilter(round, step, true,
		newMockEventProcessor(nil, processChan), true)
	eventFilter.UpdateRound(2)

	assert.Nil(t, eventFilter.Collect(new(bytes.Buffer)))

	// We should not get anything from the processChan
	timer := time.After(100 * time.Millisecond)
	select {
	case <-processChan:
		t.Fatal("processor received an obsolete event")
	case <-timer:
	}
}

func TestFlushQueueNoCheckStep(t *testing.T) {
	round := uint64(2)
	step := uint8(0) // When checkStep is false, extracted headers have a step of 0
	processChan := make(chan error, 1)
	eventFilter := newEventFilter(round, step, true,
		newMockEventProcessor(nil, processChan), false)
	eventFilter.UpdateRound(1)

	assert.Nil(t, eventFilter.Collect(new(bytes.Buffer)))
	// Update round, and flush queue to get the event
	eventFilter.UpdateRound(2)
	eventFilter.FlushQueue()

	result := <-processChan
	assert.Nil(t, result)
}

// newEventFilter simplifies the creation of an EventFilter with specific mocked
// components.
func newEventFilter(round uint64, step uint8, isMember bool,
	processor consensus.EventProcessor, checkStep bool) *consensus.EventFilter {
	return consensus.NewEventFilter(newMockHandlerFilter(round, step, []byte{}),
		consensus.NewState(), processor, checkStep)
}

func newMockEvent() wire.Event {

	mockEvent := &mocks.Event{}
	mockEvent.On("Sender").Return([]byte{})
	mockEvent.On("Equal", mock.Anything).Return(false)
	return mockEvent
}

func newMockHandlerFilter(round uint64, step uint8, pubKeyBLS []byte) consensus.EventHandler {
	var sender []byte
	mockEventHandler := &mocks.EventHandler{}
	mockEventHandler.On("Deserialize", mock.Anything).Return(newMockEvent(), nil)
	mockEventHandler.On("ExtractHeader",
		mock.MatchedBy(func(ev wire.Event) bool {
			sender = ev.Sender()
			if len(sender) == 0 {
				sender, _ = crypto.RandEntropy(32)
			}
			return true
		})).Return(func(e wire.Event) *header.Header {
		return &header.Header{
			Round:     round,
			Step:      step,
			PubKeyBLS: sender,
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
