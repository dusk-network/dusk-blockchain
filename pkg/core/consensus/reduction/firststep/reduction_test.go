package firststep

import (
	"bytes"
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
	hlp, hash := ProduceFirstStepVotes(bus, rpcBus, 50, 1*time.Second)
	// test that EventPlayer.Play has been called
	assert.Equal(t, uint8(1), hlp.Step())

	// Wait for resulting StepVotes
	svBuf := <-hlp.StepVotesChan

	// Retrieve StepVotes
	sv, err := message.UnmarshalStepVotes(&svBuf)
	assert.NoError(t, err)

	// StepVotes should be valid
	assert.NoError(t, hlp.Verify(hash, sv, 1))
	// test that the Player is PAUSED
	assert.Equal(t, consensus.PAUSED, hlp.State())
	// test that the timeout is still 1 second
	assert.Equal(t, 1*time.Second, hlp.Reducer.(*Reducer).timeOut)
}

func TestMoreSteps(t *testing.T) {
	bus, rpcBus := eventbus.New(), rpcbus.New()
	hlp, hash := ProduceFirstStepVotes(bus, rpcBus, 50, 1*time.Second)

	// Wait for resulting StepVotes
	<-hlp.StepVotesChan

	hash = hlp.NextBatch()
	svBuf := <-hlp.StepVotesChan

	// Retrieve StepVotes
	sv, err := message.UnmarshalStepVotes(&svBuf)
	assert.NoError(t, err)

	// StepVotes should be valid
	assert.NoError(t, hlp.Verify(hash, sv, 2))
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
	hlp, _ := Kickstart(bus, rpcBus, 50, timeOut)

	// Wait for resulting StepVotes
	svBuf := <-hlp.StepVotesChan
	// test we get an empty buffer
	assert.Equal(t, bytes.Buffer{}, svBuf)
	// test that EventPlayer.Play has been called
	assert.Equal(t, uint8(1), hlp.Step())
	// test that the Player is PAUSED
	assert.Equal(t, consensus.PAUSED, hlp.State())
	// test that the timeout has doubled
	assert.Equal(t, timeOut*2, hlp.Reducer.(*Reducer).timeOut)
}

func BenchmarkFirstStep(b *testing.B) {
	bus, rpcBus := eventbus.New(), rpcbus.New()
	hlp, hash := Kickstart(bus, rpcBus, 50, 1*time.Second)
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
