// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package secondstep

import (
	"context"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/require"
)

// TestSendReduction tests that the reduction step completes without problems
// and produces a StepVotesMsg in case it receives enough valid Reduction messages.
// It uses the recution common test preparation.
func TestSendReduction(t *testing.T) {
	round := uint64(1)
	messageToSpawn := 50
	hash, err := crypto.RandEntropy(32)
	require.NoError(t, err)

	timeout := time.Second

	hlp := reduction.NewHelper(messageToSpawn, timeout)
	secondStep := New(hlp.Emitter, 10*time.Second)

	// Generate second StepVotes
	svs := message.GenVotes(hash, 1, 2, hlp.ProvisionersKeys, hlp.P)
	msg := message.NewStepVotesMsg(round, hash, hlp.ThisSender, *svs[0])

	// injecting the stepVotes into secondStep
	stepFn := secondStep.Initialize(msg)

	test := reduction.PrepareSendReductionTest(hlp, stepFn)
	test(t)
}

type reductionTest struct {
	batchEvents       func(*reduction.Helper) chan message.Message
	testResultFactory consensus.TestCallback
	testStep          func(*testing.T, consensus.Phase)
}

func initiateTableTest(timeout time.Duration, hash []byte, round uint64, step uint8) map[string]reductionTest {
	return map[string]reductionTest{
		"HappyPath": {
			batchEvents: func(hlp *reduction.Helper) chan message.Message {
				evChan := make(chan message.Message, hlp.Nr)

				// creating a batch of Reduction events
				batch := hlp.Spawn(hash, round, step)
				for _, ev := range batch {
					evChan <- message.New(topics.Reduction, ev)
				}
				return evChan
			},

			testResultFactory: func(require *require.Assertions, _ consensus.InternalPacket, streamer *eventbus.GossipStreamer) {
				_, err := streamer.Read()
				require.NoError(err)

				for {
					tpcs := streamer.SeenTopics()
					for _, tpc := range tpcs {
						if tpc == topics.Agreement {
							return
						}
					}
					streamer.Read()
				}
			},

			// testing that the timeout remained the same after a successful run
			testStep: func(t *testing.T, step consensus.Phase) {
				r := step.(*Phase)

				require.Equal(t, r.TimeOut, timeout)
			},
		},

		"Timeout": {
			// no need to create events as we are testing timeouts
			batchEvents: func(hlp *reduction.Helper) chan message.Message {
				return make(chan message.Message, hlp.Nr)
			},

			// no agreement should be sent at the end of a failing second step reduction
			testResultFactory: func(require *require.Assertions, _ consensus.InternalPacket, streamer *eventbus.GossipStreamer) {
				wrongChan := make(chan struct{}, 1)
				go func() {
					for i := 0; ; i++ {
						if i == 2 {
							break
						}
						tpcs := streamer.SeenTopics()
						for _, tpc := range tpcs {
							if tpc == topics.Agreement {
								wrongChan <- struct{}{}
								return
							}
						}
						streamer.Read()
					}
				}()
				// 200 milliseconds should be plenty to receive an Agreement,
				// especially since this happens at the end of the
				// reduction step
				c := time.After(200 * time.Millisecond)
				select {
				case <-wrongChan:
					require.FailNow("unexpected Agreement message")
				case <-c:
					return
				}
			},

			// testing that the timeout doubled
			testStep: func(t *testing.T, step consensus.Phase) {
				r := step.(*Phase)
				require.Equal(t, r.TimeOut, 2*timeout)
			},
		},
	}
}

func TestSecondStepReduction(t *testing.T) {
	step := uint8(2)
	round := uint64(1)
	messageToSpawn := 50

	hash, err := crypto.RandEntropy(32)
	require.NoError(t, err)

	timeout := time.Second

	table := initiateTableTest(timeout, hash, round, step)
	for name, ttest := range table {
		streamer := eventbus.NewGossipStreamer(protocol.TestNet)
		streamListener := eventbus.NewStreamListener(streamer)

		// creating the Helper
		hlp := reduction.NewHelper(messageToSpawn, timeout)
		// wiring the Gossip streamer to capture the gossiped messages
		id := hlp.EventBus.Subscribe(topics.Gossip, streamListener)

		// running the subtest
		t.Run(name, func(t *testing.T) {
			queue := consensus.NewQueue()

			// setting up the message channel with test-specific messages in it
			evChan := ttest.batchEvents(hlp)

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

			// Generate second StepVotes
			svs := message.GenVotes(hash, 1, 2, hlp.ProvisionersKeys, hlp.P)
			msg := message.NewStepVotesMsg(round, hash, hlp.ThisSender, *svs[0])

			testPhase := consensus.NewTestPhase(t, ttest.testResultFactory, streamer)
			secondStepReduction.SetNext(testPhase)

			// injecting the stepVotes into secondStep
			secondStepReduction.Initialize(msg)

			runTestCallback := secondStepReduction.Run(ctx, queue, evChan, r, step)
			// testing the status of the step
			ttest.testStep(t, secondStepReduction)
			// here the tests are performed on the result of the step
			_ = runTestCallback.Run(ctx, queue, evChan, r, step+1)
		})

		hlp.EventBus.Unsubscribe(topics.Gossip, id)
	}
}
