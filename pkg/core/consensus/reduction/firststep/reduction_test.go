// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package firststep

import (
	"context"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/stretchr/testify/require"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

// TestSendReduction tests that the reduction step completes without problems
// and produces a StepVotesMsg in case it receives enough valid Reduction messages
// It uses the reduction common test preparation.
func TestSendReduction(t *testing.T) {
	hlp := reduction.NewHelper(50, time.Second)
	_, db := lite.CreateDBConnection()
	step := New(nil, hlp.Emitter, hlp.ProcessCandidateVerificationRequest, 10*time.Second, db, nil)
	scoreMsg := consensus.MockNewBlockMsg(t, nil)
	// injecting the result of the Selection step
	stepFn := step.Initialize(scoreMsg.Payload().(message.NewBlock))
	test := reduction.PrepareSendReductionTest(hlp, stepFn)
	test(t)
}

type reductionTest struct {
	batchEvents       func() chan message.Message
	testResultFactory consensus.TestCallback
	testStep          func(*testing.T, consensus.Phase)
}

func initiateTableTest(hlp *reduction.Helper, timeout time.Duration, hash []byte, round uint64, step uint8) map[string]reductionTest {
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

			testResultFactory: func(require *require.Assertions, packet consensus.InternalPacket, _ *eventbus.GossipStreamer, aChan chan message.Message) {
				require.NotNil(packet)

				stepVoteMessage := packet.(message.StepVotesMsg)
				require.False(stepVoteMessage.IsEmpty())

				// Retrieve StepVotes
				require.Equal(hash, stepVoteMessage.State().BlockHash)

				// StepVotes should be valid
				require.NoError(hlp.Verify(hash, stepVoteMessage.StepVotes, round, step))
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
			testResultFactory: func(require *require.Assertions, packet consensus.InternalPacket, _ *eventbus.GossipStreamer, aChan chan message.Message) {
				require.NotNil(packet)
				stepVoteMessage := packet.(message.StepVotesMsg)
				require.True(stepVoteMessage.IsEmpty())
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
			queue := consensus.NewQueue()
			// create the helper
			// setting up the message channel with predefined messages in it
			evChan := ttest.batchEvents()

			// injecting the test phase in the reduction step
			testPhase := consensus.NewTestPhase(t, ttest.testResultFactory, nil, nil)
			_, db := lite.CreateDBConnection()
			firstStepReduction := New(testPhase, hlp.Emitter, hlp.ProcessCandidateVerificationRequest, timeout, db, nil)

			// injecting the result of the Selection step
			msg := consensus.MockNewBlockMsg(t, &header.Header{BlockHash: hash})
			firstStepReduction.Initialize(msg.Payload().(message.NewBlock))

			// running the reduction step
			ctx := context.Background()
			r := consensus.RoundUpdate{
				Round: round,
				P:     *hlp.P,
				Hash:  hash,
				Seed:  hash,
			}

			runTestCallback := firstStepReduction.Run(ctx, queue, evChan, r, step)
			// testing the status of the step
			ttest.testStep(t, firstStepReduction)
			// here the tests are performed on the result of the step
			_ = runTestCallback.Run(ctx, queue, evChan, r, step+1)
		})
	}
}
