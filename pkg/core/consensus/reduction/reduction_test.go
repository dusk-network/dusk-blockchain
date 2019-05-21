package reduction

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
	"time"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

var timeOut = 200 * time.Millisecond

// func init() {
// log.SetLevel(log.DebugLevel)
// }

func mockConfig(t *testing.T) func() {

	storeDir, err := ioutil.TempDir(os.TempDir(), "reduction_test")
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

func TestReduction(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

	// send a hash to start reduction
	hash, _ := crypto.RandEntropy(32)
	committeeMock := mockCommittee(2, true, true)
	k, _ := user.NewRandKeys()

	eventBus, streamer := helper.CreateGossipStreamer()
	broker := LaunchReducer(eventBus, committeeMock, k, timeOut)
	// listen for outgoing votes of either kind, so we can verify they are being
	// sent out properly.

	// update round
	consensus.UpdateRound(eventBus, 1)

	// here we try to force the selection message to ALWAYS come after the round update
	waitForRoundUpdate(broker)
	sendSelection(hash, eventBus)

	// send mocked events until we get a result from the outgoingAgreement channel
	go func() {
		for count := 0; count < 50; count++ {
			ev := mockBlockEventBuffer(broker.ctx.state.Round(), broker.ctx.state.Step(), hash)
			eventBus.Publish(string(topics.Reduction), ev)
			time.Sleep(1 * time.Millisecond)
		}
	}()

	timer := time.AfterFunc(1*time.Second, func() {
		t.Fail()
	})

	for i := 0; i < 3; i++ {
		if _, err := streamer.Read(); err != nil {
			t.Fatal(err)
		}
	}

	timer.Stop()
}

func TestNoPublishingIfNotInCommittee(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

	// send a hash to start reduction
	hash, _ := crypto.RandEntropy(32)
	eventBus, streamer := helper.CreateGossipStreamer()
	committeeMock := mockCommittee(2, true, false)
	k, _ := user.NewRandKeys()

	broker := LaunchReducer(eventBus, committeeMock, k, timeOut)

	// update round
	consensus.UpdateRound(eventBus, 1)

	// here we try to force the selection message to ALWAYS come after the round update
	waitForRoundUpdate(broker)
	sendSelection(hash, eventBus)

	// send mocked events until we get a result from the outgoingAgreement channel
	go func() {
		for count := 0; count < 50; count++ {
			ev := mockBlockEventBuffer(broker.ctx.state.Round(), broker.ctx.state.Step(), hash)
			eventBus.Publish(string(topics.Reduction), ev)
			time.Sleep(1 * time.Millisecond)
		}
	}()

	// Try to read from the stream, and see if we get any reduction messages from
	// ourselves.
	go func() {
		for {
			if _, err := streamer.Read(); err != nil {
				t.Fatal(err)
			}

			for _, topic := range streamer.SeenTopics() {
				if topic == topics.Agreement {
					t.Fail()
				}
			}
		}
	}()

	timer := time.NewTimer(1 * time.Second)
	// if we dont get anything after a second, we can assume nothing was published.
	<-timer.C
}

func TestReductionTimeout(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

	eb, streamer := helper.CreateGossipStreamer()
	committeeMock := mockCommittee(2, true, true)
	k, _ := user.NewRandKeys()

	broker := LaunchReducer(eb, committeeMock, k, timeOut)

	// update round
	consensus.UpdateRound(eb, 1)

	// send a hash to start reduction
	hash, _ := crypto.RandEntropy(32)

	// here we try to force the selection message to ALWAYS come after the round update
	waitForRoundUpdate(broker)
	sendSelection(hash, eb)

	timer := time.After(1 * time.Second)
	<-timer

	stopChan := make(chan struct{})
	time.AfterFunc(1*time.Second, func() {
		seenTopics := streamer.SeenTopics()
		for _, topic := range seenTopics {
			if topic == topics.Agreement {
				t.Fail()
			}
		}

		stopChan <- struct{}{}
	})

	go func() {
		for {
			if _, err := streamer.Read(); err != nil {
				t.Fatal(err)
			}
		}
	}()

	<-stopChan

}

func waitForRoundUpdate(broker *broker) {
	for {
		if broker.ctx.state.Round() == 0 {
			time.Sleep(3 * time.Millisecond)
			continue
		}

		break
	}
}

func sendSelection(hash []byte, eventBus *wire.EventBus) {
	bestScoreBuf := mockSelectionEventBuffer(hash)
	eventBus.Publish(msg.BestScoreTopic, bestScoreBuf)
}

func extractTopic(buf *bytes.Buffer) [topics.Size]byte {
	var bf [topics.Size]byte
	b := make([]byte, topics.Size)
	_, _ = buf.Read(b)
	copy(bf[:], b[:])
	return bf
}
