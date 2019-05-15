package consensus_test

import (
	"bytes"
	"errors"
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

func TestAccumulation(t *testing.T) {
	// Make an accumulator that has a quorum of 2
	accumulator := consensus.NewAccumulator(newMockHandlerAccumulator(nil, 2, "foo", true), consensus.NewAccumulatorStore())
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
		errors.New("verification failed"), 2, "foo", true), consensus.NewAccumulatorStore())
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
	accumulator := consensus.NewAccumulator(newMockHandlerAccumulator(nil, 2, "foo", false), consensus.NewAccumulatorStore())
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

func newMockHandlerAccumulator(verifyErr error, quorum int, identifier string,
	isMember bool) consensus.AccumulatorHandler {
	var sender []byte
	mockEventHandler := &mocks.EventHandler{}
	mockEventHandler.On("Verify", mock.Anything).Return(verifyErr)
	mockEventHandler.On("NewEvent").Return(newMockEvent())
	mockEventHandler.On("Unmarshal", mock.Anything, mock.Anything).Return(nil)
	mockEventHandler.On("ExtractHeader",
		mock.MatchedBy(func(ev wire.Event) bool {
			sender = ev.Sender()
			if len(sender) == 0 {
				sender, _ = crypto.RandEntropy(32)
			}
			return true
		})).Return(func(e wire.Event) *header.Header {
		return &header.Header{
			Round:     1,
			Step:      1,
			PubKeyBLS: sender,
		}
	})
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
