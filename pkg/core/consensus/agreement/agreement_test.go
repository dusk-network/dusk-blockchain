package agreement_test

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// Test that the agreement component emits a round update on startup.
func TestInitBroker(t *testing.T) {
	committeeMock, k := agreement.MockCommittee(2, true, 2)
	bus := wire.NewEventBus()
	roundChan := consensus.InitRoundUpdate(bus)

	go agreement.Launch(bus, committeeMock, k[0])
	time.Sleep(200 * time.Millisecond)
	init := make([]byte, 8)
	binary.LittleEndian.PutUint64(init, 1)
	bus.Publish(msg.InitializationTopic, bytes.NewBuffer(init))

	round := <-roundChan
	assert.Equal(t, uint64(1), round)
}

// Test the accumulation of agreement events. It should result in the agreement component
// publishing a round update.
func TestBroker(t *testing.T) {
	committeeMock, keys := agreement.MockCommittee(2, true, 2)
	eb, roundChan := initAgreement(committeeMock)

	hash, _ := crypto.RandEntropy(32)
	eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 1, keys))
	eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 1, keys))

	round := <-roundChan
	assert.Equal(t, uint64(2), round)
}

// Test that the agreement component does not emit a round update if it doesn't get
// the desired amount of events.
func TestNoQuorum(t *testing.T) {
	committeeMock, keys := agreement.MockCommittee(3, true, 3)
	eb, roundChan := initAgreement(committeeMock)
	hash, _ := crypto.RandEntropy(32)
	eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 1, keys))
	eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 1, keys))

	select {
	case <-roundChan:
		assert.FailNow(t, "not supposed to get a round update without reaching quorum")
	case <-time.After(100 * time.Millisecond):
		// all good
	}
}

// Test that events, which contain a sender that is unknown to the committee, are skipped.
func TestSkipNoMember(t *testing.T) {
	committeeMock, keys := agreement.MockCommittee(1, false, 2)
	eb, roundChan := initAgreement(committeeMock)
	hash, _ := crypto.RandEntropy(32)
	eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 1, keys))

	select {
	case <-roundChan:
		assert.FailNow(t, "not supposed to get a round update without reaching quorum")
	case <-time.After(100 * time.Millisecond):
		// all good
	}
}

// Test that the agreement component properly sends out an Agreement message, upon
// receiving a ReductionResult event.
func TestSendAgreement(t *testing.T) {
	committeeMock, _ := agreement.MockCommittee(3, true, 3)
	eb, _ := initAgreement(committeeMock)

	streamer := helper.NewSimpleStreamer()
	eb.SubscribeStream(string(topics.Gossip), streamer)
	eb.RegisterPreprocessor(string(topics.Gossip), processing.NewGossip(protocol.TestNet))

	// Initiate the sending of an agreement message
	hash, _ := crypto.RandEntropy(32)
	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64(buf, binary.LittleEndian, 1); err != nil {
		t.Fatal(err)
	}

	if _, err := buf.ReadFrom(reduction.MockVoteSetBuffer(hash, 1, 2, 10)); err != nil {
		t.Fatal(err)
	}

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

// Launch the agreement component, and consume the initial round update that gets emitted.
func initAgreement(c committee.Foldable) (wire.EventBroker, <-chan uint64) {
	bus := wire.NewEventBus()
	roundChan := consensus.InitRoundUpdate(bus)
	k, _ := user.NewRandKeys()
	go agreement.Launch(bus, c, k)
	time.Sleep(200 * time.Millisecond)
	init := make([]byte, 8)
	binary.LittleEndian.PutUint64(init, 1)
	bus.Publish(msg.InitializationTopic, bytes.NewBuffer(init))

	// we remove the pre-processors here that the Launch function adds, so the mocked
	// buffers can be deserialized properly
	bus.RemoveAllPreprocessors(string(topics.Agreement))
	// we need to discard the first update since it is triggered directly as it is supposed to update the round to all other consensus compoenents
	<-roundChan
	return bus, roundChan
}
