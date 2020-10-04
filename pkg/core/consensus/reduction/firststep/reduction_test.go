package firststep

import (
	"context"
	"fmt"
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

type redTest struct {
	timeout           time.Duration
	batchEvents       func(*Helper, []byte) chan message.Message
	testResultFactory func(*Helper, []byte) consensus.TestCallback
	testStep          func(*testing.T, consensus.Phase)
}

var redtest = map[string]redTest{
	"HappyPath": redTest{
		timeout: 10 * time.Second,

		batchEvents: func(hlp *Helper, hash []byte) chan message.Message {
			evChan := make(chan message.Message, hlp.Nr)

			// creating a batch of Reduction events
			batch := hlp.Spawn(hash, round, step)
			for _, ev := range batch {
				evChan <- message.New(topics.Reduction, ev)
			}
			return evChan
		},

		testResultFactory: func(hlp *Helper, hash []byte) consensus.TestCallback {
			return func(require *require.Assertions, packet consensus.InternalPacket) error {
				require.NotNil(packet)

				if stepVoteMessage, ok := packet.(message.StepVotesMsg); ok {
					require.False(stepVoteMessage.IsEmpty())

					// Retrieve StepVotes
					require.Equal(hash, stepVoteMessage.State().BlockHash)

					// StepVotes should be valid
					require.NoError(hlp.Verify(hash, stepVoteMessage.StepVotes, round, step))

					return nil
				}
				return fmt.Errorf("Unexpected not-nil packet: %v", packet)
			}
		},

		// testing that the timeout remained the same after a successful run
		testStep: func(t *testing.T, step consensus.Phase) {
			r := step.(*Phase)
			require.Equal(t, r.timeOut, 10*time.Second)
		},
	},

	"Timeout": redTest{
		timeout: 100 * time.Millisecond,

		// no need to create events as we are testing timeouts
		batchEvents: func(hlp *Helper, hash []byte) chan message.Message {
			return make(chan message.Message, hlp.Nr)
		},

		// the result of the test should be empty step votes
		testResultFactory: func(hlp *Helper, hash []byte) consensus.TestCallback {
			return func(require *require.Assertions, packet consensus.InternalPacket) error {
				require.NotNil(packet)

				if stepVoteMessage, ok := packet.(message.StepVotesMsg); ok {
					require.True(stepVoteMessage.IsEmpty())
					return nil
				}
				return fmt.Errorf("Unexpected not-nil packet: %v", packet)
			}
		},

		// testing that the timeout doubled
		testStep: func(t *testing.T, step consensus.Phase) {
			r := step.(*Phase)
			require.Equal(t, r.timeOut, 200*time.Millisecond)
		},
	},
}

func TestFirstStepReduction(t *testing.T) {
	step := uint8(2)
	round := uint64(1)
	messageToSpawn := 50
	hash, err := crypto.RandEntropy(32)
	require.NoError(t, err)

	for name, ttest := range redtest {

		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			queue := consensus.NewQueue()
			// create the helper
			hlp := NewHelper(messageToSpawn, ttest.timeout)
			// setting up the message channel with predefined messages in it
			evChan := ttest.batchEvents(hlp, hash)

			// injecting the test phase in the reduction step
			testPhase := consensus.NewTestPhase(t, ttest.testResultFactory(hlp, hash))
			firstStepReduction := New(testPhase, hlp.Emitter, ttest.timeout)

			// injecting the result of the Selection step
			msg := consensus.MockScoreMsg(t, &header.Header{BlockHash: hash})
			firstStepReduction.Fn(msg.Payload().(message.Score))

			// running the reduction step
			ctx := context.Background()
			r := consensus.RoundUpdate{
				Round: round,
				P:     *hlp.P,
				Hash:  hash,
				Seed:  hash,
			}

			runTestCallback, err := firstStepReduction.Run(ctx, queue, evChan, r, step)
			require.NoError(err)
			// testing the status of the step
			ttest.testStep(t, firstStepReduction)
			// here the tests are performed on the result of the step
			_, err = runTestCallback(ctx, queue, evChan, r, step+1)
			// hopefully with no error
			require.NoError(err)
		})
	}
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
