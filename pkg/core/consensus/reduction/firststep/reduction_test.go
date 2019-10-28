package firststep

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestFirstStep(t *testing.T) {
	bus, rpcBus := eventbus.New(), rpcbus.New()
	hlp, hash := ProduceFirstStepVotes(bus, rpcBus, 50)

	// Wait for resulting StepVotes
	svBuf := <-hlp.StepVotesChan

	// Retrieve StepVotes
	sv, err := agreement.UnmarshalStepVotes(&svBuf)
	assert.NoError(t, err)

	// StepVotes should be valid
	assert.NoError(t, hlp.Verify(hash, sv, 1))
	// test that EventPlayer.Forward has been called
	assert.Equal(t, uint8(2), hlp.Step())
	// test that the Player is PAUSED
	assert.Equal(t, reduction.PAUSED, hlp.State)
}

func TestMoreSteps(t *testing.T) {
	bus, rpcBus := eventbus.New(), rpcbus.New()
	hlp, hash := ProduceFirstStepVotes(bus, rpcBus, 50)

	// Wait for resulting StepVotes
	<-hlp.StepVotesChan

	hash = hlp.NextBatch()
	svBuf := <-hlp.StepVotesChan

	// Retrieve StepVotes
	sv, err := agreement.UnmarshalStepVotes(&svBuf)
	assert.NoError(t, err)

	// StepVotes should be valid
	assert.NoError(t, hlp.Verify(hash, sv, 2))
	// test that EventPlayer.Forward has been called
	assert.Equal(t, uint8(3), hlp.Step())
	// test that the Player is PAUSED
	assert.Equal(t, reduction.PAUSED, hlp.State)
}

func TestFirstStepTimeOut(t *testing.T) {
	bus, rpcBus := eventbus.New(), rpcbus.New()
	hlp, _ := Kickstart(bus, rpcBus, 50)

	// Wait for resulting StepVotes
	svBuf := <-hlp.StepVotesChan
	// test we get an empty buffer
	assert.Equal(t, bytes.Buffer{}, svBuf)
	// test that EventPlayer.Forward has been called
	assert.Equal(t, uint8(2), hlp.Step())
	// test that the Player is PAUSED
	assert.Equal(t, reduction.PAUSED, hlp.State)
}

func BenchmarkFirstStep(b *testing.B) {
	bus, rpcBus := eventbus.New(), rpcbus.New()
	hlp, hash := Kickstart(bus, rpcBus, 50)
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
