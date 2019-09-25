package agreement_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-wallet/key"
	"github.com/stretchr/testify/assert"
)

// Test the accumulation of agreement events. It should result in the agreement component
// publishing a round update.
func TestBroker(t *testing.T) {
	p, keys := consensus.MockProvisioners(3)
	eb, winningHashChan := initAgreement(keys[0])
	eb.Publish(msg.RoundUpdateTopic, consensus.MockRoundUpdateBuffer(1, p, nil))

	hash, _ := crypto.RandEntropy(32)
	for i := 0; i < 3; i++ {
		eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 1, keys, p.CreateVotingCommittee(1, 1, 3)))
	}

	winningHash := <-winningHashChan
	assert.Equal(t, hash, winningHash.Bytes())
}

// Test that the agreement component does not emit a round update if it doesn't get
// the desired amount of events.
func TestNoQuorum(t *testing.T) {
	p, keys := consensus.MockProvisioners(3)
	eb, winningHashChan := initAgreement(keys[0])
	hash, _ := crypto.RandEntropy(32)
	eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 1, keys, p.CreateVotingCommittee(1, 1, 3)))
	eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 1, keys, p.CreateVotingCommittee(1, 1, 3)))

	select {
	case <-winningHashChan:
		assert.FailNow(t, "not supposed to get a round update without reaching quorum")
	case <-time.After(100 * time.Millisecond):
		// all good
	}
}

// Test that events, which contain a sender that is unknown to the committee, are skipped.
func TestSkipNoMember(t *testing.T) {
	p, keys := consensus.MockProvisioners(3)
	eb, winningHashChan := initAgreement(keys[0])
	hash, _ := crypto.RandEntropy(32)
	eb.Publish(string(topics.Agreement), agreement.MockAgreement(hash, 1, 1, keys, p.CreateVotingCommittee(1, 1, 3)))

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
	p, k := consensus.MockProvisioners(3)
	eb, _ := initAgreement(k[0])
	eb.Publish(msg.RoundUpdateTopic, consensus.MockRoundUpdateBuffer(1, p, nil))

	streamer := helper.NewSimpleStreamer()
	eb.SubscribeStream(string(topics.Gossip), streamer)
	eb.RegisterPreprocessor(string(topics.Gossip), processing.NewGossip(protocol.TestNet))

	// Initiate the sending of an agreement message
	hash, _ := crypto.RandEntropy(32)
	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, 1); err != nil {
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
func initAgreement(k key.ConsensusKeys) (eventbus.Broker, <-chan *bytes.Buffer) {
	bus := eventbus.New()
	winningHashChan := make(chan *bytes.Buffer, 1)
	bus.Subscribe(msg.WinningBlockHashTopic, winningHashChan)
	go agreement.Launch(bus, k)
	time.Sleep(200 * time.Millisecond)
	bus.Publish(msg.RoundUpdateTopic, consensus.MockRoundUpdateBuffer(1, nil, nil))

	// we remove the pre-processors here that the Launch function adds, so the mocked
	// buffers can be deserialized properly
	bus.RemoveAllPreprocessors(string(topics.Agreement))
	return bus, winningHashChan
}
