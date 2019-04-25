package agreement

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func TestInitBroker(t *testing.T) {
	committeeMock := mockCommittee(2, true)
	bus := wire.NewEventBus()
	roundChan := consensus.InitRoundUpdate(bus)

	LaunchAgreement(bus, committeeMock, 1)

	round := <-roundChan
	assert.Equal(t, uint64(1), round)
}

func TestBroker(t *testing.T) {
	committeeMock := mockCommittee(2, true)
	_, broker, roundChan := initAgreement(committeeMock)

	hash, _ := crypto.RandEntropy(32)
	e1 := MockAgreementBuf(hash, 1, 1, 2)
	e2 := MockAgreementBuf(hash, 1, 1, 2)
	_ = broker.filter.Collect(e1)
	_ = broker.filter.Collect(e2)

	round := <-roundChan
	assert.Equal(t, uint64(2), round)
}

func TestNoQuorum(t *testing.T) {
	committeeMock := mockCommittee(3, true)
	_, broker, roundChan := initAgreement(committeeMock)
	hash, _ := crypto.RandEntropy(32)
	_ = broker.filter.Collect(MockAgreementBuf(hash, 1, 1, 3))
	_ = broker.filter.Collect(MockAgreementBuf(hash, 1, 1, 3))

	select {
	case <-roundChan:
		assert.FailNow(t, "not supposed to get a round update without reaching quorum")
	case <-time.After(100 * time.Millisecond):
		// all good
	}

	_ = broker.filter.Collect(MockAgreementBuf(hash, 1, 1, 3))
	round := <-roundChan
	assert.Equal(t, uint64(2), round)
}

func TestSkipNoMember(t *testing.T) {
	committeeMock := mockCommittee(1, false)
	_, broker, roundChan := initAgreement(committeeMock)
	hash, _ := crypto.RandEntropy(32)
	_ = broker.filter.Collect(MockAgreementBuf(hash, 1, 1, 1))

	select {
	case <-roundChan:
		assert.FailNow(t, "not supposed to get a round update without reaching quorum")
	case <-time.After(100 * time.Millisecond):
		// all good
	}
}

func TestAgreement(t *testing.T) {
	committeeMock := mockCommittee(1, true)
	bus, _, roundChan := initAgreement(committeeMock)
	hash, _ := crypto.RandEntropy(32)
	bus.Publish(string(topics.Agreement), MarshalOutgoing(MockAgreementBuf(hash, 1, 1, 1)))

	round := <-roundChan
	assert.Equal(t, uint64(2), round)

	bus.Publish(string(topics.Agreement), MarshalOutgoing(MockAgreementBuf(hash, 1, 1, 1)))

	select {
	case <-roundChan:
		assert.FailNow(t, "Previous round messages should not trigger a phase update")
	case <-time.After(100 * time.Millisecond):
		// all well
		return
	}
}

func initAgreement(c committee.Committee) (wire.EventBroker, *broker, <-chan uint64) {
	bus := wire.NewEventBus()
	roundChan := consensus.InitRoundUpdate(bus)
	broker := LaunchAgreement(bus, c, 1)
	// we need to discard the first update since it is triggered directly as it is supposed to update the round to all other consensus compoenents
	<-roundChan
	return bus, broker, roundChan
}
