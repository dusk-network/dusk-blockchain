package reduction_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

var timeOut = 200 * time.Millisecond

// Test that the reduction phase works properly in the standard conditions.
func TestReduction(t *testing.T) {
	eventBus, streamer, k := launchReductionTest(true)

	// Because round updates are asynchronous (sent through a channel), we wait
	// for a bit to let the broker update its round.
	time.Sleep(200 * time.Millisecond)
	// send a hash to start reduction
	hash, _ := crypto.RandEntropy(32)
	sendSelection(1, hash, eventBus)

	// send mocked events until we get a result from the outgoingAgreement channel
	sendReductionBuffers(2, k, hash, 1, 1, eventBus)
	sendReductionBuffers(2, k, hash, 1, 2, eventBus)

	timer := time.AfterFunc(1*time.Second, func() {
		t.Fatal("")
	})

	for i := 0; i < 2; i++ {
		if _, err := streamer.Read(); err != nil {
			t.Fatal(err)
		}
	}

	timer.Stop()
}

// Test that the reducer does not send any messages when it is not part of the committee.
func TestNoPublishingIfNotInCommittee(t *testing.T) {
	eventBus, streamer, _ := launchReductionTest(false)

	// Because round updates are asynchronous (sent through a channel), we wait
	// for a bit to let the broker update its round.
	time.Sleep(200 * time.Millisecond)
	// send a hash to start reduction
	hash, _ := crypto.RandEntropy(32)
	sendSelection(1, hash, eventBus)

	// Try to read from the stream, and see if we get any reduction messages from
	// ourselves.
	go func() {
		for {
			_, err := streamer.Read()
			assert.NoError(t, err)
			// HACK: what's the point?
			t.Fatal("")
		}
	}()

	timer := time.NewTimer(1 * time.Second)
	// if we dont get anything after a second, we can assume nothing was published.
	<-timer.C
}

// Test that timeouts in the reduction phase result in proper behavior.
func TestReductionTimeout(t *testing.T) {
	eb, streamer, _ := launchReductionTest(true)

	// send a hash to start reduction
	hash, _ := crypto.RandEntropy(32)

	// Because round updates are asynchronous (sent through a channel), we wait
	// for a bit to let the broker update its round.
	time.Sleep(200 * time.Millisecond)
	sendSelection(1, hash, eb)

	timer := time.After(1 * time.Second)
	<-timer

	stopChan := make(chan struct{})
	time.AfterFunc(1*time.Second, func() {
		seenTopics := streamer.SeenTopics()
		for _, topic := range seenTopics {
			if topic == topics.Agreement {
				t.Fatal("")
			}
		}

		stopChan <- struct{}{}
	})

	<-stopChan
}

func launchCandidateVerifier(failVerification bool) {
	r := <-wire.VerifyCandidateBlockChan
	if failVerification {
		r.ErrChan <- errors.New("verification failed")
	} else {
		r.RespChan <- bytes.Buffer{}
	}
}

func launchReductionTest(inCommittee bool) (*wire.EventBus, *helper.SimpleStreamer, user.Keys) {
	eb, streamer := helper.CreateGossipStreamer()
	committeeMock := reduction.MockCommittee(2, inCommittee)
	k, _ := user.NewRandKeys()
	rpcBus := wire.NewRPCBus()
	launchReduction(eb, committeeMock, k, timeOut, rpcBus)
	// update round
	consensus.UpdateRound(eb, 1)

	return eb, streamer, k
}

func TestTimeOutVariance(t *testing.T) {
	eb, _, _ := launchReductionTest(true)

	// subscribe to reduction results
	resultChan := make(chan *bytes.Buffer, 1)
	eb.Subscribe(msg.ReductionResultTopic, resultChan)

	// Wait a bit for the round update to go through
	time.Sleep(200 * time.Millisecond)

	// measure the time it takes for reduction to time out
	start := time.Now()
	// send a hash to start reduction
	eb.Publish(msg.BestScoreTopic, nil)
	go launchCandidateVerifier(false)

	// wait for reduction to finish
	<-resultChan
	elapsed1 := time.Now().Sub(start)

	// timer should now have doubled
	start = time.Now()
	eb.Publish(msg.BestScoreTopic, nil)
	// set up another goroutine for verification
	go launchCandidateVerifier(false)

	// wait for reduction to finish
	<-resultChan
	elapsed2 := time.Now().Sub(start)

	// elapsed1 * 2 should be roughly the same as elapsed2
	assert.InDelta(t, elapsed1.Seconds()*2, elapsed2.Seconds(), 0.1)

	// update round
	consensus.UpdateRound(eb, 2)

	// Wait a bit for the round update to go through
	time.Sleep(200 * time.Millisecond)

	start = time.Now()
	// send a hash to start reduction
	eb.Publish(msg.BestScoreTopic, nil)
	// set up another goroutine for verification
	go launchCandidateVerifier(false)

	// wait for reduction to finish
	<-resultChan
	elapsed3 := time.Now().Sub(start)

	// elapsed1 and elapsed3 should be roughly the same
	assert.InDelta(t, elapsed1.Seconds(), elapsed3.Seconds(), 0.05)
}

// Convenience function, which launches the reduction component and removes the
// preprocessors for testing purposes (bypassing the republisher and the validator).
// This ensures proper handling of mocked Reduction events.
func launchReduction(eb *wire.EventBus, committee reduction.Reducers, k user.Keys, timeOut time.Duration, rpcBus *wire.RPCBus) {
	reduction.Launch(eb, committee, k, timeOut, rpcBus)
	eb.RemoveAllPreprocessors(string(topics.Reduction))
}

func sendReductionBuffers(amount int, k user.Keys, hash []byte, round uint64, step uint8,
	eventBus *wire.EventBus) {
	for i := 0; i < amount; i++ {
		ev := reduction.MockReductionBuffer(k, hash, round, step)
		eventBus.Publish(string(topics.Reduction), ev)
		time.Sleep(1 * time.Millisecond)
	}
}

func sendSelection(round uint64, hash []byte, eventBus *wire.EventBus) {
	bestScoreBuf := selection.MockSelectionEventBuffer(round, hash)
	eventBus.Publish(msg.BestScoreTopic, bestScoreBuf)
}
