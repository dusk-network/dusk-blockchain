package agreement

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

// This test checks whether there is a race condition between the agreement component's round updates,
// and the sending of agreement events.
func TestAgreementRace(t *testing.T) {
	eb := eventbus.New()

	// Sadly, we can not use the mocked committee for this test.
	// Our agreement events and reduction events need to be just like they are in production.
	// So, let's create a proper committee.
	committeeSize := 50
	p, k := consensus.MockProvisioners(committeeSize)
	broker := newBroker(eb, k[0])
	seed, _ := crypto.RandEntropy(33)
	hash, _ := crypto.RandEntropy(32)
	broker.updateRound(consensus.RoundUpdate{1, *p, nil, seed, hash})
	go broker.Listen()

	eb.RegisterPreprocessor(string(topics.Gossip), processing.NewGossip(protocol.TestNet))
	// We need to catch the outgoing agreement message
	// Let's add a SimpleStreamer to the eventbus handlers
	streamer := helper.NewSimpleStreamer()
	eb.SubscribeStream(string(topics.Gossip), streamer)

	// Create an agreement message, and update the round concurrently. If the bug has not been fixed,
	// this should cause our step counter to be off in the next round.
	// First, we make a voteset for the `sendAgreement` call.
	ru := reduction.NewUnMarshaller()
	events := CreateCommitteeVoteSet(p, k, hash, committeeSize, 1, 1)
	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, 1); err != nil {
		t.Fatal(err)
	}

	if err := ru.MarshalVoteSet(buf, events); err != nil {
		t.Fatal(err)
	}

	// Then, we send a random agreement event to the broker, to update the round.
	// Note that we start this goroutine before trying to aggregate the voteset,
	// as it takes a short while to start up.
	go func() {
		eb.Publish(msg.RoundUpdateTopic, consensus.MockRoundUpdateBuffer(2, p, nil))
	}()

	// Now we try to aggregate the reduction events created earlier. The round update should coincide
	// with the aggregation.
	eb.Publish(msg.ReductionResultTopic, buf)

	doneChan := make(chan []byte, 1)
	go readEvent(t, doneChan, streamer)

	select {
	case <-doneChan:
	case <-time.After(1 * time.Second):
		// Make sure the goroutine dies
		eb.Stream(string(topics.Gossip), bytes.NewBufferString("pippopippopippopippo"))
		<-doneChan
	}

	// Let's now create a reduction vote set with the previous public keys
	var aevBuf *bytes.Buffer

	// Because the sortition now allows multiple extraction,
	// we can not guarantee we are in the committee at any
	// given point. So, we will keep looping until we are.
	for i := 1; ; i += 2 {
		buf := new(bytes.Buffer)
		hash, _ = crypto.RandEntropy(32)
		events := CreateCommitteeVoteSet(p, k, hash, committeeSize, 2, uint8(i))
		if err := encoding.WriteUint64LE(buf, 2); err != nil {
			t.Fatal(err)
		}

		if err := ru.MarshalVoteSet(buf, events); err != nil {
			t.Fatal(err)
		}

		eb.Publish(msg.ReductionResultTopic, buf)

		doneChan := make(chan []byte, 1)
		go readEvent(t, doneChan, streamer)

		select {
		case aevBytes := <-doneChan:
			aevBuf = bytes.NewBuffer(aevBytes)
		case <-time.After(1 * time.Second):
			// Make sure the goroutine dies
			eb.Stream(string(topics.Gossip), bytes.NewBufferString("pippopippopippopippo"))
			<-doneChan
			continue
		}

		break
	}

	// Remove Ed25519 stuff first
	// 64 for signature + 32 for pubkey = 96
	edBytes := make([]byte, 96)
	if _, err := aevBuf.Read(edBytes); err != nil {
		t.Fatal(err)
	}

	unmarshaller := NewUnMarshaller()
	aev, err := unmarshaller.Deserialize(aevBuf)
	if err != nil {
		t.Fatal(err)
	}

	// Now, verify this event. The test should fail here if the bug has not been fixed.
	if err := broker.handler.Verify(aev); err != nil {
		t.Fatal(err)
	}
}

func readEvent(t *testing.T, doneChan chan []byte, streamer *helper.SimpleStreamer) {
	bs, err := streamer.Read()
	if err != nil {
		t.Fatal(err)
	}

	doneChan <- bs
}

func TestStress(t *testing.T) {
	// Setup stuff
	committeeSize := 10
	bus := eventbus.New()
	p, k := consensus.MockProvisioners(committeeSize)
	broker := newBroker(bus, k[0])
	bus.RemoveAllPreprocessors(string(topics.Agreement))
	seed, _ := crypto.RandEntropy(33)
	hash, _ := crypto.RandEntropy(32)
	broker.updateRound(consensus.RoundUpdate{1, *p, nil, seed, hash})

	// Do 10 agreement cycles
	for i := 1; i <= 10; i++ {
		hash, _ := crypto.RandEntropy(32)

		// Blast the filter with many more events than quorum, to see if anything sneaks in
		go func(i int) {
			for j := 0; j < committeeSize; j++ {
				bus.Publish(string(topics.Agreement), MockAgreement(hash, uint64(i), 1, k, p))
			}
		}(i)

		// Make sure the events we got back are all from the proper round
		evs := <-broker.filter.Accumulator.CollectedVotesChan
		for _, ev := range evs {
			assert.Equal(t, uint64(i), ev.(*Agreement).Round)
		}

		broker.updateRound(consensus.RoundUpdate{uint64(i + 1), *p, nil, seed, hash})
	}
}

func TestIncrementOnNilVoteSet(t *testing.T) {
	var k []user.Keys
	for i := 0; i < 50; i++ {
		keys, _ := user.NewRandKeys()
		k = append(k, keys)
	}
	bus := eventbus.New()
	broker := newBroker(bus, k[0])
	broker.filter.UpdateRound(1)
	go broker.Listen()

	// Wait a bit for the round update to go through
	time.Sleep(200 * time.Millisecond)

	// Run `sendAgreement` with a `voteSet` that has no votes, by publishing a
	// ReductionResult message.
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes, 1)
	bus.Publish(msg.ReductionResultTopic, bytes.NewBuffer(roundBytes))

	// Wait a bit for the message to go through
	time.Sleep(200 * time.Millisecond)

	// Now, the step counter should be at 2
	assert.Equal(t, uint8(2), broker.state.Step())
}
