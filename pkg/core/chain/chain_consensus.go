// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package chain

import (
	"context"
	"sync/atomic"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
)

// Terms and convention
//
// Consensus spin is a loop that performs Consensus
// algorithm per a single round.
//
// Consensus loop is a for-cycle that deals with
// start/stop-ing (lifecycle) of Consensus spin for each round.

// StartConsensus will start the consensus loop. It can be halted at any point by
// sending a signal through the `stopConsensus` channel (`StopConsensus`
// as exposed by the `Ledger` interface).
func (c *Chain) StartConsensus() {
	ctx, cancel := context.WithCancel(c.ctx)
	defer cancel()

	id := c.stampLoop()
	defer log.WithField("id", id).Info("consensus_loop terminated")

	// Start consensus outside of the goroutine first, so that we can ensure
	// it is fully started up while the mutex is still held.
	winnerChan := make(chan consensus.Results, 1)
	if err := c.asyncSpin(ctx, winnerChan); err != nil {
		log.WithError(err).Error("spin exited with error")
		return
	}

	// Trigger Consensus Loop
	for {
		select {
		case candidate := <-winnerChan:
			block, err := candidate.Blk, candidate.Err
			if err != nil {
				// Most likely a context cancellation, but could also be a reaching
				// of maximum steps.
				log.WithError(err).Error("consensus exited with error")
				return
			}

			c.lock.Lock()

			if block.IsEmpty() || block.Header.Height != c.tip.Header.Height+1 {
				log.WithField("height", block.Header.Height).Debugln("discarding consensus result")
				c.lock.Unlock()
				return
			}

			if err = c.acceptSuccessiveBlock(block, config.KadcastInitialHeight); err != nil {
				log.WithError(err).Error("block acceptance failed")
				c.lock.Unlock()
				return
			}

			if err := c.asyncSpin(ctx, winnerChan); err != nil {
				log.WithError(err).Error("starting consensus failed")
				c.lock.Unlock()
				return
			}

			c.lock.Unlock()
		case <-c.stopConsensusChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// StopConsensus will send a non-blocking signal to `stopConsensusChan` to
// kill the consensus goroutine.
func (c *Chain) StopConsensus() {
	select {
	case c.stopConsensusChan <- struct{}{}:
	// If there is nobody listening on the other end, it could very well be that
	// `acceptConsensusResults` is attempting to take control of the mutex.
	// In this instance, we can forego the channel send here, as the release of
	// the mutex will result in a bump to the tip height, making the consensus
	// results obsolete, and ending the lifetime of that goroutine.
	default:
	}
}

// asyncSpin initiates and triggers consensus spin in a new goroutine.
func (c *Chain) asyncSpin(ctx context.Context, winnerChan chan consensus.Results) error {
	ru := c.getRoundUpdate()

	log.WithField("round", ru.Round).
		WithField("prov_size", ru.P.Set.Len()).Info("start consensus spin")

	if c.loop == nil {
		panic("no consensus loop present")
	}

	scr, agr, err := c.loop.CreateStateMachine(c.db, config.ConsensusTimeOut, c.VerifyCandidateBlock)
	if err != nil {
		log.WithError(err).Error("could not create consensus state machine")
		return err
	}

	go func() {
		winnerChan <- c.loop.Spin(ctx, scr, agr, ru)
	}()

	return nil
}

// stampLoop increments loop counter and assings ID to loop.
// This could help on detecting dead-locked, too-long-run or multiple loops.
func (c *Chain) stampLoop() uint64 {
	correlateID := atomic.AddUint64(&c.id, 1)
	log.WithField("id", correlateID).Info("start consensus_loop")

	return correlateID
}
