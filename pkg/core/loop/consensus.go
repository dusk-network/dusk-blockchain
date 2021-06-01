// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package loop

import (
	"context"
	"errors"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/blockgenerator"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction/firststep"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction/secondstep"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "consensus loop")

// ErrMaxStepsReached is triggered when the consensus loop reaches the maximum
// amount of steps without reaching an Agreement. This means that the network
// is highly asynchronous or we are under attack.
var ErrMaxStepsReached = errors.New("consensus reached max number of steps without moving forward")

// Consensus is the state machine that runs the steps of consensus. Rather than
// trying to coordinate the various steps, it lets them execute and indicate
// which should be the following status, until completion.
// Each round is run separately and in a synchronous, blocking fashion (except
// the Agreement which should be run asynchronously by design).
type Consensus struct {
	*consensus.Emitter
	*candidate.Requestor

	pubKey *keys.PublicKey

	eventQueue *consensus.Queue
	roundQueue *consensus.Queue

	agreementChan chan message.Message
	eventChan     chan message.Message
}

// CreateStateMachine creates and link the steps in the consensus. It is kept separated from
// consensus.New so to ease mocking the consensus up when testing.
func CreateStateMachine(e *consensus.Emitter, db database.DB, consensusTimeOut time.Duration, pubKey *keys.PublicKey, verifyFn consensus.CandidateVerificationFunc, requestor *candidate.Requestor) (consensus.Phase, consensus.Controller, error) {
	generator, err := blockgenerator.New(e, pubKey, db)
	if err != nil {
		// This error means (in all cases) that there are no bid values present
		// in the db, meaning that this node is not a block generator. We will
		// not return the error, but we will log it.
		lg.WithError(err).Warnln("starting consensus loop without block generator")
	}

	selectionStep := CreateInitialStep(e, consensusTimeOut, generator, verifyFn, db, requestor)
	agreementStep := agreement.New(e, db, requestor)
	return selectionStep, agreementStep, nil
}

// CreateInitialStep creates the selection step by injecting a BlockGenerator
// interface to it.
func CreateInitialStep(e *consensus.Emitter, consensusTimeOut time.Duration, bg blockgenerator.BlockGenerator, verifyFn consensus.CandidateVerificationFunc, db database.DB, requestor *candidate.Requestor) consensus.Phase {
	redu2 := secondstep.New(e, consensusTimeOut)
	redu1 := firststep.New(redu2, e, verifyFn, consensusTimeOut, db, requestor)
	selectionStep := selection.New(redu1, bg, e, consensusTimeOut, db)

	redu2.SetNext(selectionStep)
	return selectionStep
}

// New creates a new Consensus struct. The legacy StopConsensus and RoundUpdate
// are now replaced with context cancellation and direct function call operated
// by the chain component.
func New(e *consensus.Emitter, pubKey *keys.PublicKey) *Consensus {
	// TODO: channel size should be configurable
	agreementChan := make(chan message.Message, 1000)
	eventChan := make(chan message.Message, 1000)

	// subscribe agreement phase to message.Agreement
	aChan := eventbus.NewChanListener(agreementChan)
	e.EventBus.Subscribe(topics.Agreement, aChan)

	// subscribe topics to eventChan
	evSub := eventbus.NewChanListener(eventChan)

	e.EventBus.AddDefaultTopic(topics.Reduction, topics.Score)
	e.EventBus.SubscribeDefault(evSub)

	c := &Consensus{
		Emitter:       e,
		Requestor:     candidate.NewRequestor(e.EventBus),
		pubKey:        pubKey,
		eventQueue:    consensus.NewQueue(),
		roundQueue:    consensus.NewQueue(),
		agreementChan: agreementChan,
		eventChan:     eventChan,
	}

	return c
}

// CreateStateMachine uses Consensus parameters as a shorthand for the static
// CreateStateMachine.
func (c *Consensus) CreateStateMachine(db database.DB, consensusTimeOut time.Duration, verifyFn consensus.CandidateVerificationFunc) (consensus.Phase, consensus.Controller, error) {
	return CreateStateMachine(c.Emitter, db, consensusTimeOut, c.pubKey.Copy(), verifyFn, c.Requestor)
}

//nolint:wsl
// Spin the consensus state machine. The consensus runs for the whole round
// until either a new round is produced or the node needs to re-sync. The
// Agreement loop (acting roundwise) runs concurrently with the generation-selection-reduction
// loop (acting step-wise).
// TODO: consider stopping the phase loop with a Done phase, instead of nil.
func (c *Consensus) Spin(ctx context.Context, scr consensus.Phase, ag consensus.Controller, round consensus.RoundUpdate) consensus.Results {
	// Ensure the eventQueue is emptied when the round is finished.
	defer c.eventQueue.Clear(round.Round)

	// we create two context cancelation from the same parent context. This way
	// we can let the agreement interrupt the stateMachine's loop cycle.
	// Similarly, the loop can invoke the Agreement cancelation if it throws
	// an error. Granted, the latter is less important since the parent would
	// likely invoke a cancelation when this function returns. However, the
	// small redundancy is not likely to be problematic and is an acceptable
	// overhead
	stepCtx, finalizeStep := context.WithCancel(ctx)

	agrCtx, cancelAgreement := context.WithCancel(stepCtx)
	defer cancelAgreement()

	resultsChan := make(chan consensus.Results, 1)

	// the agreement loop needs to be running until either the consensus
	// reaches a maximum amount of iterations (approx. 213 steps), or we get
	// agreements from future rounds and stopped receiving them for the current round
	// (in which case we should probably re-sync)
	go func() {
		// canceling the consensus phase loop when Agreement is done (either
		// because the parent canceled or because a consensus has been reached)
		defer finalizeStep()

		agreementLoop := ag.GetControlFn()
		results := agreementLoop(agrCtx, c.roundQueue, c.agreementChan, round)

		resultsChan <- results
	}()

	// score generation phase is the first step in the consensus
	phaseFunction := scr.Initialize(nil)
	// synchronous consensus loop keeps running until the agreement invokes
	// context.Done or the context is canceled some other way
	for step := uint8(1); ; step++ {
		phaseFunction = phaseFunction.Run(stepCtx, c.eventQueue, c.eventChan, round, step)
		// if result is nil, this round is over
		if phaseFunction == nil {
			lg.
				WithFields(log.Fields{
					"round": round.Round,
					"step":  step,
				}).
				Trace("consensus achieved")

				// Take round results from the agreement goroutine
			select {
			case results := <-resultsChan:
				return results
			default:
				return consensus.Results{Blk: block.Block{}, Err: context.Canceled}
			}
		}

		lg.
			WithFields(log.Fields{
				"round": round.Round,
				"step":  step,
				"name":  phaseFunction.String(),
			}).
			Trace("new phase")

		if config.Get().API.Enabled {
			go report(round.Round, step)
		}

		if step >= 213 {
			lg.
				WithFields(log.Fields{
					"round": round.Round,
					"step":  step,
				}).
				Error("max steps reached")
			return consensus.Results{Blk: block.Block{}, Err: ErrMaxStepsReached}
		}
	}
	// if we are here, either:
	// - agreement completed normally and we can move on to the next block
	// - agreement completed with an error and a re-synchronization needs to
	// happen
	// - caller canceled the context (we are likely re-synchronizing with
	// the network)
	// - we reached the maximum amount of steps (~213) and the consensus should
	// halt. In this case, cancel() will take care of stopping the Agreement
	// loop
}

var steps = []string{"selection", "reduction1", "reduction2"} // nolint

func report(round uint64, step uint8) {
	/*
		store := capi.GetBuntStoreInstance()
		err := store.StoreRoundInfo(round, step, "Forward", steps[(step-1)%3])
		if err != nil {
			lg.
				WithFields(log.Fields{
					"round": round,
					"step":  step,
				}).
				WithError(err).
				Error("could not save StoreRoundInfo on api db")
		}
	*/
}

// phase should start by
// - cleaning the events from the previous round
// - cleaning the events from the previous steps
// - pulling from EventQueue the events it is interested in (current
// step/category... this is a bit weird, why do we need both?)
