package reduction_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
	"time"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"
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
	committeeMock := reduction.MockCommittee(2, true)
	k, _ := user.NewRandKeys()

	eventBus, streamer := helper.CreateGossipStreamer()
	launchReduction(eventBus, committeeMock, k, timeOut)
	// listen for outgoing votes of either kind, so we can verify they are being
	// sent out properly.

	// update round
	consensus.UpdateRound(eventBus, 1)

	// Because round updates are asynchronous (sent through a channel), we wait
	// for a bit to let the broker update its round.
	time.Sleep(200 * time.Millisecond)
	sendSelection(hash, eventBus)

	// send mocked events until we get a result from the outgoingAgreement channel
	sendReductionBuffers(2, k, hash, 1, 1, eventBus)
	sendReductionBuffers(2, k, hash, 1, 2, eventBus)

	timer := time.AfterFunc(1*time.Second, func() {
		t.Fail()
	})

	for i := 0; i < 2; i++ {
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
	committeeMock := reduction.MockCommittee(2, false)
	k, _ := user.NewRandKeys()

	launchReduction(eventBus, committeeMock, k, timeOut)

	// update round
	consensus.UpdateRound(eventBus, 1)

	// Because round updates are asynchronous (sent through a channel), we wait
	// for a bit to let the broker update its round.
	time.Sleep(200 * time.Millisecond)
	sendSelection(hash, eventBus)

	// Try to read from the stream, and see if we get any reduction messages from
	// ourselves.
	go func() {
		for {
			if _, err := streamer.Read(); err != nil {
				t.Fatal(err)
			}

			t.Fail()
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
	committeeMock := reduction.MockCommittee(2, true)
	k, _ := user.NewRandKeys()

	launchReduction(eb, committeeMock, k, timeOut)

	// update round
	consensus.UpdateRound(eb, 1)

	// send a hash to start reduction
	hash, _ := crypto.RandEntropy(32)

	// Because round updates are asynchronous (sent through a channel), we wait
	// for a bit to let the broker update its round.
	time.Sleep(200 * time.Millisecond)
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

	<-stopChan
}

func launchReduction(eb *wire.EventBus, committee reduction.Reducers, k user.Keys, timeOut time.Duration) {
	reduction.Launch(eb, committee, k, timeOut)
	// remove the preprocessors for the reduction topic, to ensure proper deserialization
	// of mocked events
	eb.RegisterPreprocessor(string(topics.Reduction))
}

func sendReductionBuffers(amount int, k user.Keys, hash []byte, round uint64, step uint8,
	eventBus *wire.EventBus) {
	for i := 0; i < amount; i++ {
		ev := reduction.MockReductionBuffer(k, hash, round, step)
		eventBus.Publish(string(topics.Reduction), ev)
		time.Sleep(1 * time.Millisecond)
	}
}

func sendSelection(hash []byte, eventBus *wire.EventBus) {
	bestScoreBuf := reduction.MockSelectionEventBuffer(hash)
	eventBus.Publish(msg.BestScoreTopic, bestScoreBuf)
}

func extractTopic(buf *bytes.Buffer) [topics.Size]byte {
	var bf [topics.Size]byte
	b := make([]byte, topics.Size)
	_, _ = buf.Read(b)
	copy(bf[:], b[:])
	return bf
}
