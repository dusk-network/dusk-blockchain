// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package chain

import (
	"encoding/hex"
	"errors"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/sirupsen/logrus"
)

// allowFallback performs major verification to allow or disallow a fallback procedure.
func (c *Chain) allowFallback(b block.Block, l *logrus.Entry) error {
	// Prioritize the lowest iteration
	if b.Header.Certificate.Step > c.tip.Header.Certificate.Step {
		// We already know a winning block from lower consensus iteration.
		return errors.New("lower cetificate step")
	}

	// Fetch Previous block
	prevBlk, err := c.loader.BlockAt(c.tip.Header.Height - 1)
	if err != nil {
		return err
	}

	// Ensure block fields and certificate are valid against previous block.
	if err = c.isValidBlock(b, prevBlk, l, true); err != nil {
		return err
	}

	if b.Header.Certificate.Step == c.tip.Header.Certificate.Step {
		return errors.New("more the one winning block for the same iteration")
	}

	return nil
}

func (c *Chain) tryFallback(b block.Block) error {
	var (
		th        = c.tip.Header.Height
		stateHash []byte
		err       error

		llog = log.WithField("curr_h", th).
			WithField("curr_step", c.tip.Header.Certificate.Step).
			WithField("recv_blk_step", b.Header.Certificate.Step).
			WithField("event", "fallback")
	)

	llog.Info("initialize procedure")

	if err := c.allowFallback(b, llog); err != nil {
		return err
	}

	// Consensus fork detected. we can revert state.

	// Revert Contract Storage.
	// This will revert to the most recent finalized block.
	llog.Info("revert contract storage")

	if stateHash, err = c.proxy.Executor().Revert(c.ctx); err != nil {
		return err
	}

	llog.WithField("finalized_state_hash", hex.EncodeToString(stateHash)).
		Info("revert contract storage completed")

	// it's needed to persist otherwise we may end up having the new state
	// (after revert) inconsistant with the one that has been persisted.
	if err = c.proxy.Executor().Persist(c.ctx, stateHash); err != nil {
		return err
	}

	// Find the most recent finalized block.
	var finalized *block.Block
	err = c.db.View(func(t database.Transaction) error {
		var e error
		finalized, e = t.FetchBlockByStateRoot(th-1, stateHash)
		return e
	})

	if err != nil {
		return err
	}

	// revert blockchain from current tip to finalized block
	err = c.revertBlockchain(c.tip, finalized, llog)
	if err != nil {
		return err
	}

	llog.Info("completed")

	return nil
}

func (c *Chain) revertBlockchain(from, to *block.Block, llog *logrus.Entry) error {
	llog.WithField("from", from.Header.Height).
		WithField("to", to.Header.Height).
		Info("revert blockchain")

	err := c.db.Update(func(t database.Transaction) error {
		// Delete all non-finalized blocks
		for h := from.Header.Height; h >= to.Header.Height; h-- {
			hash, err := t.FetchBlockHashByHeight(h)
			if err != nil {
				return err
			}

			header, err := t.FetchBlockHeader(hash)
			if err != nil {
				return err
			}

			txs, err := t.FetchBlockTxs(hash)
			if err != nil {
				return err
			}

			b := block.Block{
				Header: header,
				Txs:    txs,
			}

			// resubmit txs back to mempool
			go c.resubmitTxs(txs)

			if err := t.DeleteBlock(&b); err != nil {
				return err
			}
		}

		// Store new blockchain tip and persist
		err := t.StoreBlock(to, true)
		if err != nil {
			panic(err)
		}

		c.tip = to
		return nil
	})

	return err
}

func (c *Chain) resubmitTxs(txs []transactions.ContractCall) {
	// Find diff txs between consensus-split block and new block
	for _, tx := range txs {
		if tx.Type() == transactions.Distribute {
			// Distribute tx should not be resubmitted
			continue
		}

		// transaction has not been accepted by new block then it should be resubmitted to mempool.
		if _, err := c.rpcBus.Call(topics.SendMempoolTx, rpcbus.NewRequest(tx), 5*time.Second); err != nil {
			log.WithError(err).Warn("could not resubmit txs")
		}
	}
}
