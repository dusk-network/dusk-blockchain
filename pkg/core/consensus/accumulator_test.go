package consensus_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
)

func TestAccumulation(t *testing.T) {
	// Make an accumulator that has a quorum of 2
	accumulator := consensus.NewAccumulator(newMockHandlerAccumulator(nil, 2, "foo", true))
	// Send two mock events to the accumulator
	accumulator.Process(newMockEvent())
	accumulator.Process(newMockEvent())
	// Should get something back on CollectedVotesChan
	events := <-accumulator.CollectedVotesChan
	// Should have two events
	assert.Equal(t, 2, len(events))
}

func TestFailedVerification(t *testing.T) {
	// Make an accumulator that should fail verification every time
	accumulator := consensus.NewAccumulator(newMockHandlerAccumulator(
		errors.New("verification failed"), 2, "foo", true))
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

func TestNonCommitteeEvent(t *testing.T) {
	// Make an accumulator that should fail verification every time
	accumulator := consensus.NewAccumulator(newMockHandlerAccumulator(nil, 2, "foo", false))
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
	mockCommittee.On("Quorum").Return(quorum)
	mockCommittee.On("IsMember", mock.AnythingOfType("[]uint8")).Return(isMember)
	return mockCommittee
}

type mockAccumulatorHandler struct {
	identifier string
	consensus.EventHandler
	committee.Committee
}

func newMockHandlerAccumulator(verifyErr error, quorum int, identifier string,
	isMember bool) consensus.AccumulatorHandler {
	mockHandler := &mocks.EventHandler{}
	mockHandler.On("Verify", mock.Anything).Return(verifyErr)
	return &mockAccumulatorHandler{
		EventHandler: mockHandler,
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
