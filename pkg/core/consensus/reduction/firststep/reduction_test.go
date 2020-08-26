package firststep

import (
	"bytes"
	"runtime"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestFirstStep(t *testing.T) {
	bus, rpcBus := eventbus.New(), rpcbus.New()
	hlp, hash := ProduceFirstStepVotes(bus, rpcBus, 50, 2*time.Second)
	// test that EventPlayer.Play has been called
	assert.Equal(t, uint8(1), hlp.Step())

	// Wait for resulting StepVotes
	svMsg := <-hlp.StepVotesChan

	// Retrieve StepVotes
	svm := svMsg.Payload().(message.StepVotesMsg)

	if !assert.True(t, bytes.Equal(hash, svm.State().BlockHash)) {
		t.FailNow()
	}

	// StepVotes should be valid
	if !assert.NoError(t, hlp.Verify(hash, svm.StepVotes, 1)) {
		t.FailNow()
	}
	// test that the Player is PAUSED
	assert.Equal(t, consensus.PAUSED, hlp.State())
	// test that the timeout is still 1 second
	assert.Equal(t, 2*time.Second, hlp.Reducer.(*Reducer).timeOut)
}

func TestMoreSteps(t *testing.T) {
	bus, rpcBus := eventbus.New(), rpcbus.New()
	hlp, _ := ProduceFirstStepVotes(bus, rpcBus, 50, 1*time.Second)

	// Wait for resulting StepVotes
	<-hlp.StepVotesChan

	hash := hlp.NextBatch()
	svMsg := <-hlp.StepVotesChan

	// Retrieve StepVotes
	svm := svMsg.Payload().(message.StepVotesMsg)

	// StepVotes should be valid
	assert.NoError(t, hlp.Verify(hash, svm.StepVotes, 2))
	// test that EventPlayer.Play has been called
	assert.Equal(t, uint8(2), hlp.Step())
	// test that the Player is PAUSED
	assert.Equal(t, consensus.PAUSED, hlp.State())
	// test that the timeout is still 1 second
	assert.Equal(t, 1*time.Second, hlp.Reducer.(*Reducer).timeOut)
}

func TestFirstStepTimeOut(t *testing.T) {
	bus, rpcBus := eventbus.New(), rpcbus.New()
	timeOut := 100 * time.Millisecond
	hlp, _ := KickstartConcurrent(bus, rpcBus, 50, timeOut)

	// Wait for resulting StepVotes
	svMsg := <-hlp.StepVotesChan
	svm := svMsg.Payload().(message.StepVotesMsg)

	if !assert.True(t, svm.IsEmpty()) {
		t.FailNow()
	}

	// test that EventPlayer.Play has been called
	assert.Equal(t, uint8(1), hlp.Step())
	// test that the Player is PAUSED
	assert.Equal(t, consensus.PAUSED, hlp.State())
	// test that the timeout has doubled
	assert.Equal(t, timeOut*2, hlp.Reducer.(*Reducer).timeOut)
}

func BenchmarkFirstStep(b *testing.B) {
	bus, rpcBus := eventbus.New(), rpcbus.New()
	hlp, hash := KickstartConcurrent(bus, rpcBus, 50, 1*time.Second)
	b.ResetTimer()
	b.StopTimer()
	for i := 0; i < b.N; i++ {
		evs := hlp.Spawn(hash)
		b.StartTimer()
		for _, ev := range evs {
			go hlp.Reducer.Collect(ev)
		}
		<-hlp.StepVotesChan
		b.StopTimer()
		hash, _ = crypto.RandEntropy(32)
		hlp.ActivateReduction(hash)
	}
}

// Test that we properly clean up after calling Finalize.
// TODO: trap eventual errors
func TestFinalize(t *testing.T) {
	numGRBefore := runtime.NumGoroutine()
	// Create a set of 100 agreement components, and finalize them immediately
	for i := 0; i < 100; i++ {
		bus, rpcBus := eventbus.New(), rpcbus.New()
		hlp, _ := Kickstart(bus, rpcBus, 50, 1*time.Second)

		hlp.Reducer.Finalize()
	}

	// Ensure we have freed up all of the resources associated with these components
	numGRAfter := runtime.NumGoroutine()
	// We should have roughly the same amount of goroutines
	assert.InDelta(t, numGRBefore, numGRAfter, 10.0)
}
