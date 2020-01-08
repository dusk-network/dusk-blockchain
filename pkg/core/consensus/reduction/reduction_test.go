package reduction_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction/firststep"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction/secondstep"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

// Test that we send reduction messages with the correct state in the header, in the
// event that the queue is so full that it would cause us to reach quorum by the
// time we are done processing all events.
func TestCorrectHeader(t *testing.T) {
	bus, rpcBus := eventbus.New(), rpcbus.New()
	c, hlp := wireReduction(t, bus, rpcBus)

	// Subscribe to gossip topic. We will catch the outgoing reduction votes
	// on this channel.
	gossipChan := make(chan bytes.Buffer, 2)
	bus.Subscribe(topics.Gossip, eventbus.NewChanListener(gossipChan))

	// Create reduction events from the future, enough to reach quorum for either
	// reducer.

	// Step 1
	hlp.Forward(0)
	hash, _ := crypto.RandEntropy(32)
	evs1 := hlp.Spawn(hash)
	// Step 2
	hlp.Forward(0)
	evs2 := hlp.Spawn(hash)
	evs := append(evs1, evs2...)

	// Queue the events in the coordinator
	collectEvents(t, c, evs)

	// Send a BestScore, triggering a step update and emptying the queue.
	// This should set off the two-step reduction cycle in full
	sendBestScore(t, bus, 1, 0, hash, hlp.Keys[0].BLSPubKeyBytes)

	// Collect two outgoing reduction messages. The first one should have step 1
	// in it's header, and the second should have step 2.
	r1 := <-gossipChan
	// We discard the message in the middle, as it will be an Agreement message,
	// as a result of the emptying of the queue resulting on quorum.
	<-gossipChan
	r2 := <-gossipChan

	// Retrieve headers from both reduction messages
	hdr1 := retrieveHeader(t, r1)
	hdr2 := retrieveHeader(t, r2)

	// Check correctness
	assert.Equal(t, uint8(1), hdr1.Step)
	assert.Equal(t, uint8(2), hdr2.Step)
}

func retrieveHeader(t *testing.T, r bytes.Buffer) header.Header {
	// Discard topic
	topicBuf := make([]byte, 1)
	if _, err := r.Read(topicBuf); err != nil {
		t.Fatal(err)
	}

	hdr := header.Header{}
	if err := header.Unmarshal(&r, &hdr); err != nil {
		t.Fatal(err)
	}

	return hdr
}

func sendBestScore(t *testing.T, bus *eventbus.EventBus, round uint64, step uint8, hash []byte, blsPubKey []byte) {
	hdr := header.Header{
		Round:     round,
		Step:      step,
		BlockHash: hash,
		PubKeyBLS: blsPubKey,
	}

	buf := new(bytes.Buffer)
	if err := header.Marshal(buf, hdr); err != nil {
		t.Fatal(err)
	}

	bus.Publish(topics.BestScore, buf)
}

func collectEvents(t *testing.T, c *consensus.Coordinator, evs []consensus.Event) {
	for _, ev := range evs {
		buf := new(bytes.Buffer)
		if err := header.Marshal(buf, ev.Header); err != nil {
			t.Fatal(err)
		}

		if _, err := buf.ReadFrom(&ev.Payload); err != nil {
			t.Fatal(err)
		}

		topics.Prepend(buf, topics.Reduction)
		c.CollectEvent(*buf)
	}
}

func wireReduction(t *testing.T, bus *eventbus.EventBus, rpcBus *rpcbus.RPCBus) (*consensus.Coordinator, *firststep.Helper) {
	hlp := firststep.NewHelper(bus, rpcBus, 10, 1*time.Second)
	f1 := firststep.NewFactory(bus, rpcBus, hlp.Keys[0], 1*time.Second)
	f2 := secondstep.NewFactory(bus, rpcBus, hlp.Keys[0], 1*time.Second)
	c := consensus.Start(bus, hlp.Keys[0], f1, f2)
	// Starting the coordinator
	ru := *consensus.MockRoundUpdateBuffer(1, hlp.P, nil)
	if err := c.CollectRoundUpdate(ru); err != nil {
		t.Fatal(err)
	}
	return c, hlp
}
