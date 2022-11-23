// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package chain

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
)

// RestartConsensus implements Stop and Start Consensus.
// This is a safer approach to ensure we do not duplicate Consensus loop.
func (c *Chain) RestartConsensus() error {
	c.StopConsensus()
	return c.startConsensus()
}

func (c *Chain) GetTip() uint64 {
	return c.tip.Header.Height
}

// startConsensus will start the consensus loop. It can be halted at any point by
// sending a signal through the `stopConsensus` channel (`StopConsensus`
// as exposed by the `Ledger` interface).
func (c *Chain) startConsensus() error {
	ctx, cancel := context.WithCancel(c.ctx)

	id := c.beginTrace()

	// Start consensus outside of the goroutine first, so that we can ensure
	// it is fully started up while the mutex is still held.
	winnerChan := make(chan consensus.Results, 1)
	if err := c.asyncSpin(ctx, winnerChan); err != nil {
		cancel()
		log.WithField("id", id).WithError(err).Info("consensus_loop terminated")
		return err
	}

	go func(ctx context.Context, cancel context.CancelFunc, winnerChan chan consensus.Results) {
		defer cancel()
		defer log.WithField("id", id).Info("consensus_loop terminated")

		c.acceptConsensusResults(ctx, winnerChan)
	}(ctx, cancel, winnerChan)

	return nil
}

func (c *Chain) acceptConsensusResults(ctx context.Context, winnerChan chan consensus.Results) {
	for {
		select {
		case r := <-winnerChan:
			block, err := r.Blk, r.Err
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

			if err = c.acceptSuccessiveBlock(block, nil); err != nil {
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

func (c *Chain) asyncSpin(ctx context.Context, winnerChan chan consensus.Results) error {
	ru := c.getRoundUpdate()
	consensusTimeOut := time.Duration(config.Get().Consensus.ConsensusTimeOut) * time.Second

	log.WithField("round", ru.Round).
		WithField("timeout", consensusTimeOut).
		WithField("prov_size", ru.P.Set.Len()).Debug("start consensus_spin")

	if c.loop != nil {
		scr, agr, err := c.loop.CreateStateMachine(c.db, consensusTimeOut, c.VerifyCandidateBlock, c.ExecuteStateTransition)
		if err != nil {
			log.WithError(err).Error("could not create consensus state machine")
			return err
		}

		go func() {
			winnerChan <- c.loop.Spin(ctx, scr, agr, ru)
		}()

		return nil
	}

	return errors.New("no consensus loop present")
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

// beginTrace increments loop counter and assings ID to loop.
// This will help on detecting dead-locked, too-long-run or multiple loops errors.
func (c *Chain) beginTrace() uint64 {
	correlateID := atomic.AddUint64(&c.loopID, 1)
	log.WithField("id", correlateID).Info("start consensus_loop")

	return correlateID
}
