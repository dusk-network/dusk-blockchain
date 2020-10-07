package secondstep

import (
	"context"
	"fmt"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TestSendReduction tests that the reduction step completes without problems
// and produces a StepVotesMsg in case it receives enough valid Reduction messages
// It uses the recution common test preparation
func TestSendReduction(t *testing.T) {
	round := uint64(1)
	messageToSpawn := 50
	hash, err := crypto.RandEntropy(32)
	require.NoError(t, err)

	timeout := time.Second

	hlp := reduction.NewHelper(messageToSpawn, timeout)
	secondStep := New(hlp.Emitter, 10*time.Second)

	// Generate first StepVotes
	svs := message.GenVotes(hash, 1, 2, hlp.ProvisionersKeys, hlp.P)
	msg := message.NewStepVotesMsg(round, hash, hlp.ThisSender, *svs[0])

	// injecting the stepVotes into secondStep
	stepFn := secondStep.Fn(msg)

	test := reduction.PrepareSendReductionTest(hlp, stepFn)
	test(t)
}

type reductionTest struct {
	batchEvents       func() chan message.Message
	testResultFactory consensus.TestCallback
	testStep          func(*testing.T, consensus.Phase)
}

func initiateTableTest(hlp *reduction.Helper, timeout time.Duration, hash []byte, round uint64, step uint8, savedStepVotes *message.StepVotesMsg) map[string]reductionTest {

	return map[string]reductionTest{
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

					savedStepVotes = &stepVoteMessage

					return nil
				}

				return fmt.Errorf("unexpected not-nil packet: %v", packet)
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
				return fmt.Errorf("unexpected not-nil packet: %v", packet)
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
	var savedStepVotes *message.StepVotesMsg

	table := initiateTableTest(hlp, timeout, hash, round, step, savedStepVotes)
	for name, ttest := range table {

		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			queue := consensus.NewQueue()
			// create the helper
			// setting up the message channel with predefined messages in it
			evChan := ttest.batchEvents()

			// running the reduction step
			ctx := context.Background()
			r := consensus.RoundUpdate{
				Round: round,
				P:     *hlp.P,
				Hash:  hash,
				Seed:  hash,
			}

			// spin secondStepVotes
			secondStepReduction := New(hlp.Emitter, timeout)

			round := uint64(1)
			messageToSpawn := 50
			hash, err := crypto.RandEntropy(32)
			require.NoError(err)

			timeout := time.Second

			hlp := reduction.NewHelper(messageToSpawn, timeout)

			// Generate first StepVotes
			svs := message.GenVotes(hash, 1, 2, hlp.ProvisionersKeys, hlp.P)
			msg := message.NewStepVotesMsg(round, hash, hlp.ThisSender, *svs[0])

			//scoreStep, err := score.New(nil, hlp.Emitter, blockGen)
			cb := func(ctx context.Context) (bool, error) {
				packet := ctx.Value("Packet")
				require.NotNil(t, packet)
				return true, nil
			}

			mockPhase := consensus.MockPhase(cb)
			secondStepReduction.SetNext(mockPhase)

			// injecting the stepVotes into secondStep
			secondStepReduction.Fn(msg)

			runTestCallback, err := secondStepReduction.Run(ctx, queue, evChan, r, step)
			require.NoError(err)
			// testing the status of the step
			ttest.testStep(t, secondStepReduction)
			// here the tests are performed on the result of the step
			_, err = runTestCallback(ctx, queue, evChan, r, step+1)
			// hopefully with no error
			require.NoError(err)

		})
	}
}
