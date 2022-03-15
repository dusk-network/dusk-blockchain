// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package chain

import (
	"errors"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/sirupsen/logrus"
)

// allowFallback performs major verification to allow or disallow a fallback procedure.
func (c *Chain) allowFallback(fallbackBlock block.Block, l *logrus.Entry) (*block.Block, error) {
	// Prioritize the lowest iteration
	if fallbackBlock.Header.Certificate.Step > c.tip.Header.Certificate.Step {
		return nil, errors.New("lower cetificate iteration")
	}

	// Fetch Previous block
	prevBlk, err := c.loader.BlockAt(c.tip.Header.Height - 1)
	if err != nil {
		return nil, err
	}

	// Ensure block fields and certificate are valid against previous block.
	if err = c.isValidBlock(fallbackBlock, prevBlk, l, true); err != nil {
		return nil, err
	}

	return &prevBlk, nil
}

func (c *Chain) tryFallback(blk block.Block) error {
	l := log.WithField("curr_h", c.tip.Header.Height).
		WithField("curr_step", c.tip.Header.Certificate.Step).
		WithField("fallback_step", blk.Header.Certificate.Step).
		WithField("event", "fallback")

	l.Info("initialize procedure")

	var prevBlk *block.Block
	var err error

	if prevBlk, err = c.allowFallback(blk, l); err != nil {
		return err
	}

	// Once fallback is allowed, we can revert state.

	// Revert Contract Storage.
	// It's performed before accepting the new block otherwise
	// grpc.Accept will fail.

	l.Info("revert contract storage")

	if err := c.proxy.Executor().Revert(c.ctx); err != nil {
		return err
	}

	// Revert chain tip
	l.Info("revert in-memory blockchain tip")

	oldTip := c.tip.Copy().(block.Block)
	c.tip = prevBlk

	// Revert last block
	l.Info("revert last block")

	if err := c.acceptBlock(blk, true); err != nil {
		c.tip = &oldTip
		return err
	}

	// Revert a mempool
	l.Info("revert mempool")

	c.revertMempool(&oldTip, &blk)

	// Restart consensus algorithm
	if err := c.RestartConsensus(); err != nil {
		l.WithError(err).Warn("failed to restart consensus loop")
	}

	l.Info("completed")

	return nil
}

func (c *Chain) revertMempool(oldBlk, newBlk *block.Block) {
	// Find diff txs between consensus-split block and new block
	for _, tx := range oldBlk.Txs {
		if tx.Type() == transactions.Distribute {
			// Distribute tx should not be resubmitted
			continue
		}

		h, err := tx.CalculateHash()
		if err != nil {
			break
		}

		if _, err := newBlk.Tx(h); err != nil {
			// transaction has not been accepted by new block then it should be resubmitted to mempool.
			if _, err := c.rpcBus.Call(topics.SendMempoolTx, rpcbus.NewRequest(tx), 5*time.Second); err != nil {
				log.WithError(err).Warn("could not resubmit txs")
			}
		}
	}
}
