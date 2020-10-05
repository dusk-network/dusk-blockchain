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
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

// TestSendReduction tests that the reduction step completes without problems
// and produces a StepVotesMsg in case it receives enough valid Reduction messages
func TestSendReduction(t *testing.T) {
	hlp := reduction.NewHelper(50, time.Second)

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
	batchEvents       func() chan message.Message
	testResultFactory consensus.TestCallback
	testStep          func(*testing.T, consensus.Phase)
}

func initiateTableTest(hlp *reduction.Helper, timeout time.Duration, hash []byte, round uint64, step uint8) map[string]redTest {

	return map[string]redTest{
		"HappyPath": {
			batchEvents: func() chan message.Message {
				evChan := make(chan message.Message, hlp.Nr)

				// creating a batch of Reduction events
				batch := hlp.Spawn(hash, round, step)
				for _, ev := range batch {
					evChan <- message.New(topics.Reduction, ev)
				}
				return evChan
			},

			testResultFactory: func(require *require.Assertions, packet consensus.InternalPacket) error {
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
			},

			// testing that the timeout remained the same after a successful run
			testStep: func(t *testing.T, step consensus.Phase) {
				r := step.(*Phase)
				require.Equal(t, r.TimeOut, timeout)
			},
		},

		"Timeout": {
			// no need to create events as we are testing timeouts
			batchEvents: func() chan message.Message {
				return make(chan message.Message, hlp.Nr)
			},

			// the result of the test should be empty step votes
			testResultFactory: func(require *require.Assertions, packet consensus.InternalPacket) error {
				require.NotNil(packet)

				if stepVoteMessage, ok := packet.(message.StepVotesMsg); ok {
					require.True(stepVoteMessage.IsEmpty())
					return nil
				}
				return fmt.Errorf("Unexpected not-nil packet: %v", packet)
			},

			// testing that the timeout doubled
			testStep: func(t *testing.T, step consensus.Phase) {
				r := step.(*Phase)
				require.Equal(t, r.TimeOut, 2*timeout)
			},
		},
	}
}

func TestFirstStepReduction(t *testing.T) {
	step := uint8(2)
	round := uint64(1)
	messageToSpawn := 50
	hash, err := crypto.RandEntropy(32)
	require.NoError(t, err)
	timeout := time.Second

	hlp := reduction.NewHelper(messageToSpawn, timeout)

	table := initiateTableTest(hlp, timeout, hash, round, step)
	for name, ttest := range table {

		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			queue := consensus.NewQueue()
			// create the helper
			// setting up the message channel with predefined messages in it
			evChan := ttest.batchEvents()

			// injecting the test phase in the reduction step
			testPhase := consensus.NewTestPhase(t, ttest.testResultFactory)
			firstStepReduction := New(testPhase, hlp.Emitter, timeout)

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
