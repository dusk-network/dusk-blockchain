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
	timeOut := 200 * time.Millisecond
	k, _ := user.NewRandKeys()

	eventBus, streamer := helper.CreateGossipStreamer()
	broker := LaunchReducer(eventBus, committeeMock, k, timeOut)
	// listen for outgoing votes of either kind, so we can verify they are being
	// sent out properly.

	// update round
	consensus.UpdateRound(eventBus, 1)

	// here we try to force the selection message to ALWAYS come after the round update
	for {
		if broker.ctx.state.Round() == 0 {
			time.Sleep(3 * time.Millisecond)
			continue
		}

		// now we can send the selection
		bestScoreBuf := mockSelectionEventBuffer(hash)
		eventBus.Publish(msg.BestScoreTopic, bestScoreBuf)
		break
	}

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
	eventBus := wire.NewEventBus()
	committeeMock := mockCommittee(2, true, false)
	timeOut := 200 * time.Millisecond
	k, _ := user.NewRandKeys()

	broker := LaunchReducer(eventBus, committeeMock, k, timeOut)
	// listen for outgoing votes of either kind, so we can verify they are being
	// sent out properly.
	outgoingReduction := make(chan *bytes.Buffer, 2)
	outgoingAgreement := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe(msg.OutgoingBlockReductionTopic, outgoingReduction)
	eventBus.Subscribe(msg.OutgoingBlockAgreementTopic, outgoingAgreement)

	// update round
	consensus.UpdateRound(eventBus, 1)

	// here we try to force the selection message to ALWAYS come after the round update
	for {
		if broker.ctx.state.Round() == 0 {
			time.Sleep(3 * time.Millisecond)
			continue
		}

		// now we can send the selection
		bestScoreBuf := mockSelectionEventBuffer(hash)
		eventBus.Publish(msg.BestScoreTopic, bestScoreBuf)
		break
	}

	// send mocked events until we get a result from the outgoingAgreement channel
	timer := time.After(1 * time.Second)
	count := 0
	for {
		select {
		case <-outgoingAgreement:
			t.Fatal("message should not have been sent as AmMember is set to false")
			return
		case <-timer:
			//All good
			return
		default:
			count++
			// the committee is never more than 50 nodes. It does not make much sense to hammer the reducer more than that
			if count < 50 {
				ev := mockBlockEventBuffer(broker.ctx.state.Round(), broker.ctx.state.Step(), hash)
				eventBus.Publish(string(topics.Reduction), ev)
			}
		}
	}
}

func TestReductionTimeout(t *testing.T) {
	fn := mockConfig(t)
	defer fn()

	eb, streamer := helper.CreateGossipStreamer()
	committeeMock := mockCommittee(2, true, true)
	timeOut := 200 * time.Millisecond
	k, _ := user.NewRandKeys()

	broker := LaunchReducer(eb, committeeMock, k, timeOut)

	// update round
	consensus.UpdateRound(eb, 1)

	// send a hash to start reduction
	hash, _ := crypto.RandEntropy(32)

	// here we try to force the selection message to ALWAYS come after the round update
	for {
		if broker.ctx.state.Round() == 0 {
			time.Sleep(3 * time.Millisecond)
			continue
		}

		// now we can send the selection
		bestScoreBuf := mockSelectionEventBuffer(hash)
		eb.Publish(msg.BestScoreTopic, bestScoreBuf)
		break
	}

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

func extractTopic(buf *bytes.Buffer) [topics.Size]byte {
	var bf [topics.Size]byte
	b := make([]byte, topics.Size)
	_, _ = buf.Read(b)
	copy(bf[:], b[:])
	return bf
}
