package agreement_test

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
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

	committeeMock, _ := agreement.MockCommittee(2, true, 2)
	bus := wire.NewEventBus()
	roundChan := consensus.InitRoundUpdate(bus)

	agreement.Launch(bus, committeeMock, user.Keys{}, 1)

	round := <-roundChan
	assert.Equal(t, uint64(1), round)
}

func TestBroker(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

	committeeMock, keys := agreement.MockCommittee(2, true, 2)
	eb, roundChan := initAgreement(committeeMock)

	hash, _ := crypto.RandEntropy(32)
	eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 2, keys))
	eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 2, keys))

	round := <-roundChan
	assert.Equal(t, uint64(2), round)
}

func TestNoQuorum(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

	committeeMock, keys := agreement.MockCommittee(3, true, 3)
	eb, roundChan := initAgreement(committeeMock)
	hash, _ := crypto.RandEntropy(32)
	eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 2, keys))
	eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 2, keys))

	select {
	case <-roundChan:
		assert.FailNow(t, "not supposed to get a round update without reaching quorum")
	case <-time.After(100 * time.Millisecond):
		// all good
	}

	eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 2, keys))
	round := <-roundChan
	assert.Equal(t, uint64(2), round)
}

func TestSkipNoMember(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

	committeeMock, keys := agreement.MockCommittee(1, false, 2)
	eb, roundChan := initAgreement(committeeMock)
	hash, _ := crypto.RandEntropy(32)
	eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 2, keys))

	select {
	case <-roundChan:
		assert.FailNow(t, "not supposed to get a round update without reaching quorum")
	case <-time.After(100 * time.Millisecond):
		// all good
	}
}

func TestSendAgreement(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

	committeeMock, _ := agreement.MockCommittee(3, true, 3)
	eb, _ := initAgreement(committeeMock)

	streamer := helper.NewSimpleStreamer()
	eb.SubscribeStream(string(topics.Gossip), streamer)
	eb.RegisterPreprocessor(string(topics.Gossip), peer.NewGossip(protocol.TestNet))

	// Initiate the sending of an agreement message
	hash, _ := crypto.RandEntropy(32)
	buf := reduction.MockVoteSetBuffer(hash, 1, 2, 10)
	eb.Publish(msg.ReductionResultTopic, buf)

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

func initAgreement(c committee.Foldable) (wire.EventBroker, <-chan uint64) {
	bus := wire.NewEventBus()
	roundChan := consensus.InitRoundUpdate(bus)
	k, _ := user.NewRandKeys()
	agreement.Launch(bus, c, k, 1)
	// we remove the pre-processors here that the Launch function adds, so the mocked
	// buffers can be deserialized properly
	bus.RegisterPreprocessor(string(topics.Agreement))
	// we need to discard the first update since it is triggered directly as it is supposed to update the round to all other consensus compoenents
	<-roundChan
	return bus, roundChan
}
