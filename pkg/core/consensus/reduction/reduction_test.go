package reduction_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

var timeOut = 200 * time.Millisecond

func TestStress(t *testing.T) {
	eventBus, streamer, k := launchReductionTest(true)

	// subscribe for the voteset
	voteSetChan := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe(msg.ReductionResultTopic, voteSetChan)

	// Because round updates are asynchronous (sent through a channel), we wait
	// for a bit to let the broker update its round.
	time.Sleep(200 * time.Millisecond)
	hash, _ := crypto.RandEntropy(32)
	for i := 0; i < 10; i++ {
		go launchCandidateVerifier(false)
		go func() {
			// Blast the reducer with events
			for i := 1; i <= 100; i++ {
				go sendReductionBuffers(50, k, hash, 1, uint8(i), eventBus)
			}
		}()
		sendSelection(1, hash, eventBus)

		for i := 0; i < 2; i++ {
			// discard reduction events, we dont give a fuck about those
			if _, err := streamer.Read(); err != nil {
				t.Fatal(err)
			}
		}

		voteSetBuf := <-voteSetChan
		// Remove the round bytes first
		var round uint64
		if err := encoding.ReadUint64(voteSetBuf, binary.LittleEndian, &round); err != nil {
			t.Fatal(err)
		}

		// Unmarshal votesBytes and check them for correctness
		unmarshaller := reduction.NewUnMarshaller()
		voteSet, err := unmarshaller.UnmarshalVoteSet(voteSetBuf)
		if err != nil {
			t.Fatal(err)
		}

		// Make sure we have an equal amount of votes for each step, and no events snuck in where they don't belong
		stepMap := make(map[uint8]int)
		for _, vote := range voteSet {
			stepMap[vote.(*reduction.Reduction).Step]++
		}

		assert.Equal(t, stepMap[1], stepMap[2])
	}
}

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
