package agreement

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
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

	eb.Register(topics.Gossip, processing.NewGossip(protocol.TestNet))
	// We need to catch the outgoing agreement message
	// Let's add a SimpleStreamer to the eventbus handlers

	streamer := eventbus.NewGossipStreamer()
	streamListener := eventbus.NewStreamListener(streamer)

	eb.Subscribe(topics.Gossip, streamListener)

	// Create an agreement message, and update the round concurrently. If the bug has not been fixed,
	// this should cause our step counter to be off in the next round.
	// First, we make a voteset for the `sendAgreement` call.
	ru := reduction.NewUnMarshaller()
	events := createVoteSet(t, k, hash, committeeSize, 1)
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
		eb.Publish(topics.RoundUpdate, consensus.MockRoundUpdateBuffer(2, p, nil))
	}()

	// Now we try to aggregate the reduction events created earlier. The round update should coincide
	// with the aggregation.
	eb.Publish(topics.ReductionResult, buf)

	// Read and discard the resulting event
	if _, err := streamer.Read(); err != nil {
		t.Fatal(err)
	}

	// Let's now create a reduction vote set with the previous public keys
	hash, _ = crypto.RandEntropy(32)
	events = createVoteSet(t, k, hash, committeeSize, 2)
	buf.Reset()
	if err := encoding.WriteUint64LE(buf, 2); err != nil {
		t.Fatal(err)
	}

	if err := ru.MarshalVoteSet(buf, events); err != nil {
		t.Fatal(err)
	}

	eb.Publish(topics.ReductionResult, buf)

	aevBytes, err := streamer.Read()
	if err != nil {
		t.Fatal(err)
	}

	aevBuf := bytes.NewBuffer(aevBytes)

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

func TestStress(t *testing.T) {
	// Setup stuff
	committeeSize := 10
	bus := eventbus.New()
	p, k := consensus.MockProvisioners(committeeSize)
	broker := newBroker(bus, k[0])
	bus.RemoveProcessors(topics.Agreement)
	seed, _ := crypto.RandEntropy(33)
	hash, _ := crypto.RandEntropy(32)
	broker.updateRound(consensus.RoundUpdate{1, *p, nil, seed, hash})

	// Do 10 agreement cycles
	for i := 1; i <= 10; i++ {
		hash, _ := crypto.RandEntropy(32)

		// Blast the filter with many more events than quorum, to see if anything sneaks in
		go func(i int) {
			for j := 0; j < committeeSize; j++ {
				bus.Publish(topics.Agreement, MockAgreement(hash, uint64(i), 1, k, p.CreateVotingCommittee(uint64(i), 1, committeeSize)))
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
	bus.Publish(topics.ReductionResult, bytes.NewBuffer(roundBytes))

	// Wait a bit for the message to go through
	time.Sleep(200 * time.Millisecond)

	// Now, the step counter should be at 2
	assert.Equal(t, uint8(2), broker.state.Step())
}

func createVoteSet(t *testing.T, k []user.Keys, hash []byte, size int, round uint64) (events []wire.Event) {
	for i := 1; i <= 2; i++ {
		// We should only pick a certain key once, no dupes.
		picked := make(map[int]struct{})

		// We need 75% of the committee size worth of events to reach quorum.
		for j := 0; j < int(float64(size)*0.75); j++ {
			// Key choices need to be randomized. There is no sorting in production.
			var n int
			for {
				n = rand.Intn(size)
				if _, ok := picked[n]; !ok {
					picked[n] = struct{}{}
					break
				}
			}

			ev := reduction.MockReduction(k[n], hash, round, uint8(i))
			events = append(events, ev)
		}
	}

	return
}
