package agreement

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func TestInitBroker(t *testing.T) {
	committeeMock, k := MockCommittee(2, true, 2)
	bus := wire.NewEventBus()
	roundChan := consensus.InitRoundUpdate(bus)

	LaunchAgreement(bus, committeeMock, k[0], 1)

	round := <-roundChan
	assert.Equal(t, uint64(1), round)
}

func TestBroker(t *testing.T) {
	committeeMock, keys := MockCommittee(2, true, 2)
	tc := collectAgreements(committeeMock, keys, 2)
	round := <-tc.roundChan
	assert.Equal(t, tc.round+1, round)
}

func TestNoQuorum(t *testing.T) {
	committeeMock, keys := MockCommittee(3, true, 3)
	tc := collectAgreements(committeeMock, keys, 2)

	select {
	case <-tc.roundChan:
		assert.FailNow(t, "not supposed to get a round update without reaching quorum")
	case <-time.After(100 * time.Millisecond):
		// all good
	}

	_ = tc.broker.filter.Collect(MockAgreement(tc.hash, tc.round, tc.step, keys))
	round := <-tc.roundChan
	assert.Equal(t, uint64(2), round)
}

func TestSkipNoMember(t *testing.T) {
	committeeMock, keys := MockCommittee(1, false, 2)
	tc := collectAgreements(committeeMock, keys, 1)

	select {
	case <-tc.roundChan:
		assert.FailNow(t, "not supposed to get a round update without reaching quorum")
	case <-time.After(100 * time.Millisecond):
		// all good
	}
}

func TestSendAgreement(t *testing.T) {

	committeeMock, _ := MockCommittee(3, true, 3)
	eb, broker, _ := initAgreement(committeeMock)

	streamer := helper.NewSimpleStreamer()
	eb.SubscribeStream(string(topics.Gossip), streamer)
	eb.RegisterPreprocessor(string(topics.Gossip), peer.NewGossip(protocol.TestNet))

	// Initiate the sending of an agreement message
	hash, _ := crypto.RandEntropy(32)
	buf := reduction.MockVoteSetBuffer(hash, 1, 2, 10)
	if err := broker.sendAgreement(buf); err != nil {
		panic(err)
	}

	// There should now be an agreement message in the streamer
	_, err := streamer.Read()
	if err != nil {
		t.Fatal(err)
	}

	seenTopics := streamer.SeenTopics()
	if seenTopics[0] != topics.Agreement {
		t.Fail()
	}
}

type testComponents struct {
	broker    *broker
	roundChan <-chan uint64
	hash      []byte
	round     uint64
	step      uint8
}

func collectAgreements(c committee.Foldable, k []user.Keys, nr int) *testComponents {
	_, broker, roundChan := initAgreement(c)
	hash, _ := crypto.RandEntropy(32)
	for i := 0; i < nr; i++ {
		_ = broker.filter.Collect(MockAgreement(hash, 1, 2, k))
	}
	return &testComponents{
		broker,
		roundChan,
		hash,
		uint64(1),
		uint8(2),
	}
}

func initAgreement(c committee.Foldable) (wire.EventBroker, *broker, <-chan uint64) {
	bus := wire.NewEventBus()
	roundChan := consensus.InitRoundUpdate(bus)
	k, _ := user.NewRandKeys()
	broker := LaunchAgreement(bus, c, k, 1)
	// we need to discard the first update since it is triggered directly as it is supposed to update the round to all other consensus compoenents
	<-roundChan
	return bus, broker, roundChan
}
