package agreement_test

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/committee"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

// Test the accumulation of agreement events. It should result in the agreement component
// publishing a round update.
func TestBroker(t *testing.T) {
	committeeMock, keys := agreement.MockCommittee(2, true, 2)
	eb, winningHashChan := initAgreement(committeeMock)

	hash, _ := crypto.RandEntropy(32)
	eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 1, keys))
	eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 1, keys))

	winningHash := <-winningHashChan
	assert.Equal(t, hash, winningHash.Bytes())
}

// Test that the agreement component does not emit a round update if it doesn't get
// the desired amount of events.
func TestNoQuorum(t *testing.T) {
	committeeMock, keys := agreement.MockCommittee(3, true, 3)
	eb, winningHashChan := initAgreement(committeeMock)
	hash, _ := crypto.RandEntropy(32)
	eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 1, keys))
	eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 1, keys))

	select {
	case <-winningHashChan:
		assert.FailNow(t, "not supposed to get a round update without reaching quorum")
	case <-time.After(100 * time.Millisecond):
		// all good
	}
}

// Test that events, which contain a sender that is unknown to the committee, are skipped.
func TestSkipNoMember(t *testing.T) {
	committeeMock, keys := agreement.MockCommittee(1, false, 2)
	eb, winningHashChan := initAgreement(committeeMock)
	hash, _ := crypto.RandEntropy(32)
	eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 1, keys))

	select {
	case <-winningHashChan:
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
func initAgreement(c committee.Foldable) (wire.EventBroker, <-chan *bytes.Buffer) {
	bus := wire.NewEventBus()
	winningHashChan := make(chan *bytes.Buffer, 1)
	bus.Subscribe(msg.WinningBlockHashTopic, winningHashChan)
	k, _ := user.NewRandKeys()
	go agreement.Launch(bus, c, k)
	time.Sleep(200 * time.Millisecond)
	bus.Publish(msg.RoundUpdateTopic, mockRoundUpdateBuffer(1))

	// we remove the pre-processors here that the Launch function adds, so the mocked
	// buffers can be deserialized properly
	bus.RemoveAllPreprocessors(string(topics.Agreement))
	return bus, winningHashChan
}

func mockRoundUpdateBuffer(round uint64) *bytes.Buffer {
	init := make([]byte, 8)
	binary.LittleEndian.PutUint64(init, round)
	buf := bytes.NewBuffer(init)

	members := mockMembers(1)
	user.MarshalMembers(buf, members)

	bidList := mockBidList()
	user.MarshalBidList(buf, bidList)
	return buf
}

func mockMembers(amount int) []user.Member {
	members := make([]user.Member, amount)

	for i := 0; i < amount; i++ {
		keys, _ := user.NewRandKeys()
		member := &user.Member{}
		member.PublicKeyEd = keys.EdPubKeyBytes
		member.PublicKeyBLS = keys.BLSPubKeyBytes
		member.Stakes = make([]user.Stake, 1)
		member.Stakes[0].Amount = 500
		member.Stakes[0].EndHeight = 10000
		members[i] = *member

	}

	return members
}

func mockBidList() user.BidList {
	bidList := make([]user.Bid, 1)
	xSlice, _ := crypto.RandEntropy(32)
	mSlice, _ := crypto.RandEntropy(32)
	var x [32]byte
	var m [32]byte
	copy(x[:], xSlice)
	copy(m[:], mSlice)
	bidList[0] = user.Bid{x, m, 10}
	return bidList
}
