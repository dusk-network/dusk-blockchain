package agreement

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func mockConfig(t *testing.T) func() {

	storeDir, err := ioutil.TempDir(os.TempDir(), "agreement_test")
	if err != nil {
		t.Fatal(err.Error())
	}

	r := cfg.Registry{}
	r.Performance.AccumulatorWorkers = 4
	cfg.Mock(&r)

	return func() {
		os.RemoveAll(storeDir)
	}
}

func TestInitBroker(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

	committeeMock, _ := mockCommittee(2, true, 2)
	bus := wire.NewEventBus()
	roundChan := consensus.InitRoundUpdate(bus)

	LaunchAgreement(bus, committeeMock, nil, 1)

	round := <-roundChan
	assert.Equal(t, uint64(1), round)
}

func TestBroker(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

	committeeMock, keys := mockCommittee(2, true, 2)
	_, broker, roundChan := initAgreement(committeeMock)

	hash, _ := crypto.RandEntropy(32)
	e1 := MockAgreement(hash, 1, 2, keys)
	e2 := MockAgreement(hash, 1, 2, keys)
	_ = broker.filter.Collect(e1)
	_ = broker.filter.Collect(e2)

	round := <-roundChan
	assert.Equal(t, uint64(2), round)
}

func TestNoQuorum(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

	committeeMock, keys := mockCommittee(3, true, 3)
	_, broker, roundChan := initAgreement(committeeMock)
	hash, _ := crypto.RandEntropy(32)
	_ = broker.filter.Collect(MockAgreement(hash, 1, 2, keys))
	_ = broker.filter.Collect(MockAgreement(hash, 1, 2, keys))

	select {
	case <-roundChan:
		assert.FailNow(t, "not supposed to get a round update without reaching quorum")
	case <-time.After(100 * time.Millisecond):
		// all good
	}

	_ = broker.filter.Collect(MockAgreement(hash, 1, 2, keys))
	round := <-roundChan
	assert.Equal(t, uint64(2), round)
}

func TestSkipNoMember(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

	committeeMock, keys := mockCommittee(1, false, 2)
	_, broker, roundChan := initAgreement(committeeMock)
	hash, _ := crypto.RandEntropy(32)
	_ = broker.filter.Collect(MockAgreement(hash, 1, 2, keys))

	select {
	case <-roundChan:
		assert.FailNow(t, "not supposed to get a round update without reaching quorum")
	case <-time.After(100 * time.Millisecond):
		// all good
	}
}

func initAgreement(c committee.Committee) (wire.EventBroker, *broker, <-chan uint64) {
	bus := wire.NewEventBus()
	roundChan := consensus.InitRoundUpdate(bus)
	broker := LaunchAgreement(bus, c, nil, 1)
	// we need to discard the first update since it is triggered directly as it is supposed to update the round to all other consensus compoenents
	<-roundChan
	return bus, broker, roundChan
}
