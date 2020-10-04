package firststep

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/stretchr/testify/require"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

// TestSendReduction tests that the reduction step completes without problems
// and produces a StepVotesMsg in case it receives enough valid Reduction messages
func TestSendReduction(t *testing.T) {
	hlp := NewHelper(50, time.Second)

	streamer := eventbus.NewGossipStreamer(protocol.TestNet)
	hlp.EventBus.Subscribe(topics.Gossip, eventbus.NewStreamListener(streamer))

	step := New(nil, hlp.Emitter, 10*time.Second)

	msg := consensus.MockScoreMsg(t, nil)
	// injecting the result of the Selection step
	step.Fn(msg.Payload().(message.Score))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_, err := streamer.Read()
		require.NoError(t, err)
		require.Equal(t, streamer.SeenTopics()[0], topics.Reduction)
		cancel()
	}()

	evChan := make(chan message.Message, 1)
	n, err := step.Run(ctx, consensus.NewQueue(), evChan, consensus.MockRoundUpdate(uint64(1), hlp.P), uint8(2))
	require.Nil(t, n)
	require.NoError(t, err)
}

func TestFirstStep(t *testing.T) {
	queue := consensus.NewQueue()
	evChan := make(chan message.Message, 50)
	step := uint8(2)
	round := uint64(1)

	messageToSpawn := 50

	hash, err := crypto.RandEntropy(32)
	require.Nil(t, err)

	consensusTimeOut := 10 * time.Second

	hlp := NewHelper(messageToSpawn, time.Second)

	cb := func(ctx context.Context) (bool, error) {
		packet := ctx.Value("Packet")
		require.NotNil(t, packet)

		if stepVoteMessage, ok := packet.(message.StepVotesMsg); ok {
			require.False(t, stepVoteMessage.IsEmpty())

			// Retrieve StepVotes
			require.Equal(t, hash, stepVoteMessage.State().BlockHash)

			// StepVotes should be valid
			require.NoError(t, hlp.Verify(hash, stepVoteMessage.StepVotes, round, step))

			return true, nil
		}
		return false, errors.New("cb: failed to validate Score")
	}

	resultPhase := consensus.MockPhase(cb)
	firstStepReduction := New(resultPhase, hlp.Emitter, consensusTimeOut)

	msg := consensus.MockScoreMsg(t, &header.Header{BlockHash: hash})

	// injecting the result of the Selection step
	firstStepReduction.Fn(msg.Payload().(message.Score))

	ctx := context.Background()

	go func() {
		// creating a batch of Reduction events
		batch := hlp.Spawn(hash, round, step)
		for _, ev := range batch {
			evChan <- message.New(topics.Reduction, ev)
		}

	}()

	r := consensus.RoundUpdate{
		Round: uint64(1),
		P:     *hlp.P,
		Hash:  hash,
		Seed:  hash,
	}
	resultFn, err := firstStepReduction.Run(ctx, queue, evChan, r, step)
	require.NoError(t, err)
	require.NotNil(t, resultFn)

	_, err = resultFn(ctx, queue, evChan, r, step+1)

	require.Nil(t, err)

}

//func TestFirstStepTimeOut(t *testing.T) {
//	bus, rpcBus := eventbus.New(), rpcbus.New()
//	timeOut := 100 * time.Millisecond
//	hlp, _ := KickstartConcurrent(bus, rpcBus, 50, timeOut)
//
//	// Wait for resulting StepVotes
//	svMsg := <-hlp.StepVotesChan
//	svm := svMsg.Payload().(message.StepVotesMsg)
//
//	if !assert.True(t, svm.IsEmpty()) {
//		t.FailNow()
//	}
//
//	// test that EventPlayer.Play has been called
//	assert.Equal(t, uint8(1), hlp.Step())
//	// test that the Player is PAUSED
//	assert.Equal(t, consensus.PAUSED, hlp.State())
//	// test that the timeout has doubled
//	assert.Equal(t, timeOut*2, hlp.Reducer.(*Reducer).timeOut)
//}
//
//func BenchmarkFirstStep(b *testing.B) {
//	bus, rpcBus := eventbus.New(), rpcbus.New()
//	hlp, hash := KickstartConcurrent(bus, rpcBus, 50, 1*time.Second)
//	b.ResetTimer()
//	b.StopTimer()
//	for i := 0; i < b.N; i++ {
//		evs := hlp.Spawn(hash)
//		b.StartTimer()
//		for _, ev := range evs {
//			go hlp.Reducer.Collect(ev)
//		}
//		<-hlp.StepVotesChan
//		b.StopTimer()
//		hash, _ = crypto.RandEntropy(32)
//		hlp.ActivateReduction(hash)
//	}
//}
//
//// Test that we properly clean up after calling Finalize.
//// TODO: trap eventual errors
//func TestFinalize(t *testing.T) {
//	numGRBefore := runtime.NumGoroutine()
//	// Create a set of 100 agreement components, and finalize them immediately
//	for i := 0; i < 100; i++ {
//		bus, rpcBus := eventbus.New(), rpcbus.New()
//		hlp, _ := Kickstart(bus, rpcBus, 50, 1*time.Second)
//
//		hlp.Reducer.Finalize()
//	}
//
//	// Ensure we have freed up all of the resources associated with these components
//	numGRAfter := runtime.NumGoroutine()
//	// We should have roughly the same amount of goroutines
//	assert.InDelta(t, numGRBefore, numGRAfter, 10.0)
//}
