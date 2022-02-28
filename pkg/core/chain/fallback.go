// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package chain

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

func (c *Chain) tryFallback(blk block.Block) error {
	l := log.WithField("curr_h", c.tip.Header.Height).WithField("event", "fallback")
	l.Info("initialize procedure")

	// Prioritize the lowest iteration
	if blk.Header.Certificate.Step > c.tip.Header.Certificate.Step {
		l.Info("discarded")
		return nil
	}

	// Fetch Previous block
	prevBlk, err := c.loader.BlockAt(c.tip.Header.Height - 1)
	if err != nil {
		return err
	}

	oldTip := c.tip.Copy().(block.Block)
	c.tip = &prevBlk

	// Perform verify and accept block procedure
	if err := c.acceptBlock(blk, true); err != nil {
		c.tip = &oldTip
		return err
	}

	// Perform a mempool rollback
	c.rollbackMempool(&oldTip, &blk)

	if err := c.RestartConsensus(); err != nil {
		l.WithError(err).Warn("failed to restart consensus loop")
	}

	l.Info("completed")

	return nil
}

func (c *Chain) rollbackMempool(oldBlk, newBlk *block.Block) {
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
