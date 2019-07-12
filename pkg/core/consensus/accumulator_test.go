package consensus_test

import (
	"bytes"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/header"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func TestAccumulatorGoroutines(t *testing.T) {
	state := consensus.NewState()
	// Let's spawn 100 accumulators
	for i := 0; i < 100; i++ {
		// Make an accumulator that has a quorum of 2
		accumulator := consensus.NewAccumulator(newMockHandlerAccumulator(1, 1, nil, []byte{}, 2, "foo", true), consensus.NewAccumulatorStore(), state, false)
		// Set WorkerTimeOut to 1 ms
		accumulator.WorkerTimeOut = 100 * time.Millisecond
		// Create workers
		accumulator.CreateWorkers()
		go accumulator.Accumulate()
	}

	time.Sleep(3 * time.Second)
	state.Update(2)
	time.Sleep(1 * time.Second)
	// We spawned 500 goroutines, so if there is only a fraction of that left,
	// it means we are successful (concurrent tests can take up goroutines too)
	assert.True(t, runtime.NumGoroutine() <= 15)
}

// Test the accumulation of events up to Quorum. The Accumulator should return the
// events on the CollectedVotesChan once we do.
func TestAccumulation(t *testing.T) {
	// Make an accumulator that has a quorum of 2
	accumulator := consensus.NewAccumulator(newMockHandlerAccumulator(1, 1, nil, []byte{}, 2, "foo", true), consensus.NewAccumulatorStore(), consensus.NewState(), true)
	accumulator.CreateWorkers()
	go accumulator.Accumulate()
	// Send two mock events to the accumulator
	accumulator.Process(newMockEvent())
	accumulator.Process(newMockEvent())
	// Should get something back on CollectedVotesChan
	events := <-accumulator.CollectedVotesChan
	// Should have two events
	assert.Equal(t, 2, len(events))
}

// Test that events which fail verification are not stored.
func TestFailedVerification(t *testing.T) {
	// Make an accumulator that should fail verification every time
	accumulator := consensus.NewAccumulator(newMockHandlerAccumulator(1, 1,
		errors.New("verification failed"), []byte{}, 2, "foo", true), consensus.NewAccumulatorStore(), consensus.NewState(), false)
	accumulator.CreateWorkers()
	go accumulator.Accumulate()
	// Send two mock events to the accumulator
	accumulator.Process(newMockEvent())
	accumulator.Process(newMockEvent())
	// We should not get anything from the CollectedVotesChan
	timer := time.After(100 * time.Millisecond)
	select {
	case <-accumulator.CollectedVotesChan:
		t.Fatal("reached quorum when passed events should have been dropped")
	case <-timer:
	}
}

// Test that events which come from senders which are not in the committee are ignored.
func TestNonCommitteeEvent(t *testing.T) {
	// Make an accumulator that should fail verification every time
	accumulator := consensus.NewAccumulator(newMockHandlerAccumulator(1, 1, nil, []byte{}, 2, "foo", false), consensus.NewAccumulatorStore(), consensus.NewState(), false)
	accumulator.CreateWorkers()
	go accumulator.Accumulate()
	// Send two mock events to the accumulator
	accumulator.Process(newMockEvent())
	accumulator.Process(newMockEvent())
	// We should not get anything from the CollectedVotesChan
	timer := time.After(100 * time.Millisecond)
	select {
	case <-accumulator.CollectedVotesChan:
		t.Fatal("reached quorum when passed events should have been dropped")
	case <-timer:
	}
}

func newMockCommittee(quorum int, isMember bool) committee.Committee {
	mockCommittee := &mocks.Committee{}
	mockCommittee.On("Quorum", mock.Anything).Return(quorum)
	mockCommittee.On("IsMember",
		mock.AnythingOfType("[]uint8"),
		mock.AnythingOfType("uint64"),
		mock.AnythingOfType("uint8"),
	).Return(isMember)
	return mockCommittee
}

type mockAccumulatorHandler struct {
	identifier string
	consensus.EventHandler
	committee.Committee
}

func newMockHandlerAccumulator(round uint64, step uint8, verifyErr error, sender []byte, quorum int, identifier string,
	isMember bool) consensus.AccumulatorHandler {
	mockEventHandler := &mocks.EventHandler{}
	mockEventHandler.On("Deserialize", mock.Anything).Return(newMockEvent(), nil)
	mockEventHandler.On("ExtractHeader",
		mock.MatchedBy(func(ev wire.Event) bool {
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
	mockEventHandler.On("Verify", mock.Anything).Return(verifyErr)
	mockEventHandler.On("NewEvent").Return(newMockEvent())
	mockEventHandler.On("Unmarshal", mock.Anything, mock.Anything).Return(nil)
	return &mockAccumulatorHandler{
		EventHandler: mockEventHandler,
		Committee:    newMockCommittee(quorum, isMember),
		identifier:   identifier,
	}
}

// ExtractIdentifier implements the AccumulatorHandler interface.
// Will write the stored identifier on the passed buffer.
func (m *mockAccumulatorHandler) ExtractIdentifier(e wire.Event, r *bytes.Buffer) error {
	if _, err := r.Write([]byte(m.identifier)); err != nil {
		return err
	}

	return nil
}
