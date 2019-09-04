package agreement

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/committee"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

// This test checks whether there is a race condition between the agreement component's round updates,
// and the sending of agreement events.
func TestAgreementRace(t *testing.T) {
	eb := wire.NewEventBus()
	_, db := lite.CreateDBConnection()

	// Sadly, we can not use the mocked committee for this test.
	// Our agreement events and reduction events need to be just like they are in production.
	// So, let's create a proper committee.
	c := committee.NewAgreement(eb, db)
	// We make a set of 50 keys, add these as provisioners and start the agreement component.
	committeeSize := 50
	k := createProvisionerSet(t, c, committeeSize)
	broker := newBroker(eb, c, k[0])
	broker.updateRound(1)
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
	hash, _ := crypto.RandEntropy(32)
	events := createVoteSet(t, k, hash, committeeSize, 1)
	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64(buf, binary.LittleEndian, 1); err != nil {
		t.Fatal(err)
	}

	if err := ru.MarshalVoteSet(buf, events); err != nil {
		t.Fatal(err)
	}

	// Then, we send a random agreement event to the broker, to update the round.
	// Note that we start this goroutine before trying to aggregate the voteset,
	// as it takes a short while to start up.
	go func(events []wire.Event) {
		broker.filter.Accumulator.CollectedVotesChan <- []wire.Event{MockAgreementEvent(hash, 1, 1, k)}
	}(events)

	// Now we try to aggregate the reduction events created earlier. The round update should coincide
	// with the aggregation.
	eb.Publish(msg.ReductionResultTopic, buf)

	// Read and discard the resulting event
	if _, err := streamer.Read(); err != nil {
		t.Fatal(err)
	}

	// Let's now create a reduction vote set with the previous public keys
	hash, _ = crypto.RandEntropy(32)
	events = createVoteSet(t, k, hash, committeeSize, 2)

	// Now, we create our agreement event. Our step counter should be one off from the events we are aggregating.
	aevBuf, err := broker.handler.createAgreement(events, broker.state.Round(), broker.state.Step())
	eb.Stream(string(topics.Gossip), aevBuf)

	aevBytes, err := streamer.Read()
	if err != nil {
		t.Fatal(err)
	}

	aevBuf = bytes.NewBuffer(aevBytes)

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
	committeeMock, k := MockCommittee(20, true, 20)
	bus := wire.NewEventBus()
	broker := newBroker(bus, committeeMock, k[0])
	broker.filter.UpdateRound(1)
	bus.RemoveAllPreprocessors(string(topics.Agreement))

	// Wait a bit for the round update to go through
	time.Sleep(200 * time.Millisecond)

	// Do 10 agreement cycles
	for i := 1; i <= 10; i++ {
		hash, _ := crypto.RandEntropy(32)

		// Blast the filter with many more events than quorum, to see if anything sneaks in
		go func(i int) {
			for j := 0; j < 50; j++ {
				bus.Publish(string(topics.Agreement), MockAgreement(hash, uint64(i), 1, k))
			}
		}(i)

		// Make sure the events we got back are all from the proper round
		evs := <-broker.filter.Accumulator.CollectedVotesChan
		for _, ev := range evs {
			assert.Equal(t, uint64(i), ev.(*Agreement).Round)
		}

		broker.filter.UpdateRound(uint64(i + 1))
	}
}

func createProvisionerSet(t *testing.T, c *committee.Agreement, size int) (k []user.Keys) {
	for i := 0; i < size; i++ {
		keys, _ := user.NewRandKeys()
		buf := new(bytes.Buffer)
		if err := encoding.Write256(buf, keys.EdPubKeyBytes); err != nil {
			t.Fatal(err)
		}

		if err := encoding.WriteVarBytes(buf, keys.BLSPubKeyBytes); err != nil {
			t.Fatal(err)
		}

		if err := encoding.WriteUint64(buf, binary.LittleEndian, 500); err != nil {
			t.Fatal(err)
		}

		if err := encoding.WriteUint64(buf, binary.LittleEndian, 0); err != nil {
			t.Fatal(err)
		}

		if err := encoding.WriteUint64(buf, binary.LittleEndian, 10000); err != nil {
			t.Fatal(err)
		}

		if err := c.AddProvisioner(buf); err != nil {
			t.Fatal(err)
		}
		k = append(k, keys)
	}

	return
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
