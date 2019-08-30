package agreement

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/committee"
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
)

func TestWrongCommittee(t *testing.T) {
	// Sadly, we can not use the mocked committee for this test
	// So, let's create one
	eb := wire.NewEventBus()
	_, db := lite.CreateDBConnection()
	c := committee.NewAgreement(eb, db)
	// We make a set of 50 keys, add these as provisioners and start the agreement component with key zero
	committeeSize := 50
	var k []user.Keys
	for i := 0; i < committeeSize; i++ {
		keys, _ := user.NewRandKeys()
		buf := new(bytes.Buffer)
		_ = encoding.Write256(buf, keys.EdPubKeyBytes)
		_ = encoding.WriteVarBytes(buf, keys.BLSPubKeyBytes)
		_ = encoding.WriteUint64(buf, binary.LittleEndian, 500)
		_ = encoding.WriteUint64(buf, binary.LittleEndian, 0)
		_ = encoding.WriteUint64(buf, binary.LittleEndian, 10000)
		if err := c.AddProvisioner(buf); err != nil {
			t.Fatal(err)
		}
		k = append(k, keys)
	}

	broker := newBroker(eb, c, k[0])
	broker.updateRound(1)

	eb.RegisterPreprocessor(string(topics.Gossip), processing.NewGossip(protocol.TestNet))
	// We need to catch the outgoing agreement message
	// Let's add a SimpleStreamer to the eventbus handlers
	streamer := helper.NewSimpleStreamer()
	eb.SubscribeStream(string(topics.Gossip), streamer)

	// Increment the step counter on the agreement component
	broker.state.IncrementStep()

	// Let's now create a reduction vote set with the previous public keys
	hash, _ := crypto.RandEntropy(32)

	var events []wire.Event
	for i := 1; i <= 2; i++ {
		// We should only pick a certain key once, no dupes
		picked := make(map[int]struct{})

		// We do 38 votes per step, 75% of 50
		for j := 0; j < 38; j++ {
			// Key choices need to be randomized. There is no sorting in production
			var n int
			for {
				n = rand.Intn(50)
				if _, ok := picked[n]; !ok {
					picked[n] = struct{}{}
					break
				}
			}

			ev := reduction.MockReduction(k[n], hash, 1, uint8(i))
			events = append(events, ev)
		}
	}

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

	// Now, verify this event.
	if err := broker.handler.Verify(aev); err != nil {
		t.Fatal(err)
	}
}
