package agreement_test

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	logrus.SetLevel(logrus.TraceLevel)
}

// Test the accumulation of agreement events. It should result in the agreement component
// publishing a round update.
func TestAgreement(t *testing.T) {
	nr := 50
	_, hlp := wireAgreement(nr)
	hash, _ := crypto.RandEntropy(32)
	vc := hlp.P.CreateVotingCommittee(1, 1, nr)
	for i := 0; i < nr; i++ {
		aev := agreement.MockWire(hash, 1, 1, hlp.Keys, vc, i)
		hlp.Bus.Publish(topics.Agreement, aev)
	}

	res := <-hlp.WinningHashChan
	assert.Equal(t, hash, res.Bytes())
}

func wireAgreement(nrProvisioners int) (*consensus.Coordinator, *agreement.Helper) {
	eb := eventbus.New()
	h := agreement.NewHelper(eb, nrProvisioners)
	factory := agreement.NewFactory(eb, h.Keys[0])
	coordinator := consensus.Start(eb, h.Keys[0], factory)
	// starting up the coordinator
	ru := *consensus.MockRoundUpdateBuffer(1, h.P, nil)
	if err := coordinator.CollectRoundUpdate(ru); err != nil {
		panic(err)
	}
	// we need to remove annoying ED25519 verification or the Republisher
	eb.RemoveAllProcessors()
	return coordinator, h
}

/*
// Test that the agreement component does not emit a round update if it doesn't get
// the desired amount of events.
func TestNoQuorum(t *testing.T) {
	p, keys := consensus.MockProvisioners(3)
	eb, winningHashChan := initAgreement(keys[0])
	hash, _ := crypto.RandEntropy(32)
	eb.Publish(topics.Agreement, agreement.MockAgreement(hash, 1, 1, keys, p.CreateVotingCommittee(1, 1, 3)))
	eb.Publish(topics.Agreement, agreement.MockAgreement(hash, 1, 1, keys, p.CreateVotingCommittee(1, 1, 3)))

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
	eb.Publish(topics.Agreement, agreement.MockAgreement(hash, 1, 1, keys, p.CreateVotingCommittee(1, 1, 3)))

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
	eb.Publish(topics.RoundUpdate, consensus.MockRoundUpdateBuffer(1, p, nil))

	streamer := eventbus.NewGossipStreamer(protocol.TestNet)
	streamListener := eventbus.NewStreamListener(streamer)
	eb.Subscribe(topics.Gossip, streamListener)
	eb.Register(topics.Gossip, processing.NewGossip(protocol.TestNet))

	// Initiate the sending of an agreement message
	hash, _ := crypto.RandEntropy(32)
	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, 1); err != nil {
		t.Fatal(err)
	}

	if _, err := buf.ReadFrom(reduction.MockVoteSetBuffer(hash, 1, 2, 10)); err != nil {
		t.Fatal(err)
	}

	eb.Publish(topics.ReductionResult, buf)

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
func initAgreement(k user.Keys) (eventbus.Broker, <-chan bytes.Buffer) {
	bus := eventbus.New()
	winningHashChan := make(chan bytes.Buffer, 1)
	chanListener := eventbus.NewChanListener(winningHashChan)
	bus.Subscribe(topics.WinningBlockHash, chanListener)

	go agreement.Launch(bus, k)
	time.Sleep(200 * time.Millisecond)
	bus.Publish(topics.RoundUpdate, consensus.MockRoundUpdateBuffer(1, nil, nil))

	// we remove the pre-processors here that the Launch function adds, so the mocked
	// buffers can be deserialized properly
	bus.RemoveProcessors(topics.Agreement)
	return bus, winningHashChan
}
*/
