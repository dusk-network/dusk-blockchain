package agreement

import (
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

type MockHandler struct {
	amMember  bool
	isMember  bool
	committee user.VotingCommittee
	quorum    int
	verify    bool
}

func (m *MockHandler) AmMember(round uint64, step uint8) bool {
	return m.amMember
}

func (m *MockHandler) IsMember(pubKeyBLS []byte, round uint64, step uint8) bool {
	return m.isMember
}

func (m *MockHandler) VotesFor(pubKeyBLS []byte, round uint64, step uint8) int {
	return 1
}

func (m *MockHandler) Committee(round uint64, step uint8) user.VotingCommittee {
	return m.committee
}

func (m *MockHandler) Quorum(uint64) int {
	return m.quorum
}

// Verify checks the signature of the set.
func (m *MockHandler) Verify(ev message.Agreement) error {
	if !m.verify {
		return errors.New("dog told me to")
	}
	return nil
}

func TestAccumulatorStop(t *testing.T) {
	hdlr := &MockHandler{true, true, user.VotingCommittee{}, 2, true}
	accumulator := newAccumulator(hdlr, 100)
	go accumulator.Accumulate()

	time.Sleep(3 * time.Second)
	accumulator.Stop()
	time.Sleep(time.Second)
	assert.True(t, runtime.NumGoroutine() <= 15)
}

// Test the accumulation of events up to Quorum. The Accumulator should return the
// events on the CollectedVotesChan once we do.
func TestAccumulation(t *testing.T) {
	// Make an accumulator that has a quorum of 2
	hdlr := &MockHandler{true, true, user.VotingCommittee{}, 2, true}
	accumulator := newAccumulator(hdlr, 4)
	go accumulator.Accumulate()

	createAgreement := newAggroFactory(10)

	// Send two mock events to the accumulator
	accumulator.Process(createAgreement(1, 1, 1))
	accumulator.Process(createAgreement(1, 1, 2))
	// Should get something back on CollectedVotesChan
	events := <-accumulator.CollectedVotesChan
	// Should have two events
	assert.Equal(t, 2, len(events))
}

func TestStop(t *testing.T) {
	// Make an accumulator that has a quorum of 3
	hdlr := &MockHandler{true, true, user.VotingCommittee{}, 3, true}
	accumulator := newAccumulator(hdlr, 4)
	go accumulator.Accumulate()

	createAgreement := newAggroFactory(10)

	// Send two mock events to the accumulator
	accumulator.Process(createAgreement(1, 1, 1))
	accumulator.Process(createAgreement(1, 1, 2))
	accumulator.Stop()
	accumulator.Process(createAgreement(1, 1, 3))

	// Should NOT get something back on CollectedVotesChan
	select {
	case <-accumulator.CollectedVotesChan:
		assert.FailNow(t, "accumulator should not have collected votes")
	case <-time.After(50 * time.Millisecond):
		// all good
	}
}

// Test that events which fail verification are not stored.
func TestFailedVerification(t *testing.T) {
	// Make an accumulator that has a quorum of 2 and fails verification
	hdlr := &MockHandler{true, true, user.VotingCommittee{}, 3, false}
	accumulator := newAccumulator(hdlr, 4)
	go accumulator.Accumulate()

	createAgreement := newAggroFactory(10)

	// Send two mock events to the accumulator
	accumulator.Process(createAgreement(1, 1, 1))
	accumulator.Process(createAgreement(1, 1, 2))
	// We should not get anything from the CollectedVotesChan
	timer := time.After(100 * time.Millisecond)
	select {
	case <-accumulator.CollectedVotesChan:
		assert.FailNow(t, "reached quorum when passed events should have been dropped")
	case <-timer:
	}
}

// Test that events which fail verification are not stored.
func TestNotInCommittee(t *testing.T) {
	// Make an accumulator that has a quorum of 1 and is not in the committee
	hdlr := &MockHandler{true, false, user.VotingCommittee{}, 1, false}
	accumulator := newAccumulator(hdlr, 4)
	go accumulator.Accumulate()

	createAgreement := newAggroFactory(10)

	// Send two mock events to the accumulator
	accumulator.Process(createAgreement(1, 1, 1))
	// We should not get anything from the CollectedVotesChan
	timer := time.After(100 * time.Millisecond)
	select {
	case <-accumulator.CollectedVotesChan:
		assert.FailNow(t, "reached quorum when passed events should have been dropped")
	case <-timer:
	}
}

/*
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

type mockAccumulatorHandler struct {
	identifier string
	quorum     int
	isMember   bool
}

func newMockHandlerAccumulator(round uint64, step uint8, verifyErr error, sender []byte, quorum int, identifier string,
	isMember bool) consensus.AccumulatorHandler {
	mockEventHandler := &mocks.EventHandler{}
	mockEventHandler.On("Deserialize", mock.Anything).Return(newMockEvent(), nil)
	mockEventHandler.On("Verify", mock.Anything).Return(verifyErr)
	mockEventHandler.On("NewEvent").Return(newMockEvent())
	mockEventHandler.On("Unmarshal", mock.Anything, mock.Anything).Return(nil)
	return mockAccumulatorHandler{
		EventHandler: mockEventHandler,
		identifier:   identifier,
		quorum:       quorum,
		isMember:     isMember,
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

func (m *mockAccumulatorHandler) Quorum() int {
	return m.quorum
}

func (m *mockAccumulatorHandler) IsMember(pubKeyBLS []byte, round uint64, step uint8) bool {
	return m.isMember
}
*/

func newAggroFactory(provisionersNr int) func(uint64, uint8, int) message.Agreement {
	hash, _ := crypto.RandEntropy(32)
	p, ks := consensus.MockProvisioners(provisionersNr)

	return func(round uint64, step uint8, idx int) message.Agreement {
		a := MockAgreementEvent(hash, round, step, ks, p, idx)
		return *a
	}
}
