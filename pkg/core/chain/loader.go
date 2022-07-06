// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package chain

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/verifiers"
)

const (
	// SanityCheckHeight is the suggested amount of blocks to check when
	// calling Loader.SanityCheckBlockchain.
	SanityCheckHeight uint64 = 10
)

// DBLoader performs database prefetching and sanityChecks at node startup.
type DBLoader struct {
	db database.DB

	// Unsure if the genesis block needs to be here.
	genesis *block.Block
}

// SanityCheckBlock will verify whether we have not seed the block before
// (duplicate), perform a check on the block header and verifies the coinbase
// transactions. It leaves the bulk of transaction verification to the executor
// Return nil if the sanity check passes.
func (l *DBLoader) SanityCheckBlock(prevBlock block.Block, blk block.Block) error {
	// 1. Check that we have not seen this block before
	err := l.db.View(func(t database.Transaction) error {
		_, err := t.FetchBlockExists(blk.Header.Hash)
		return err
	})

	if err != database.ErrBlockNotFound {
		if err == nil {
			err = errors.New("block already exists")
		}

		return err
	}

	if err := verifiers.CheckBlockHeader(prevBlock, blk); err != nil {
		return err
	}

	return nil
}

// NewDBLoader returns a Loader which gets the Chain Tip from the DB.
func NewDBLoader(db database.DB, genesis *block.Block) *DBLoader {
	return &DBLoader{db: db, genesis: genesis}
}

// Height returns the height of the blockchain stored in the DB.
func (l *DBLoader) Height() (uint64, error) {
	var height uint64

	err := l.db.View(func(t database.Transaction) error {
		var err error
		height, err = t.FetchCurrentHeight()
		return err
	})

	return height, err
}

// BlockAt returns the block stored at a given height.
func (l *DBLoader) BlockAt(searchingHeight uint64) (block.Block, error) {
	var blk *block.Block

	err := l.db.View(func(t database.Transaction) error {
		hash, err := t.FetchBlockHashByHeight(searchingHeight)
		if err != nil {
			return err
		}

		blk, err = t.FetchBlock(hash)
		return err
	})
	if err != nil {
		return block.Block{}, err
	}

	return *blk, err
}

// Clear the underlying DB.
func (l *DBLoader) Clear() error {
	return l.db.Update(func(t database.Transaction) error {
		return t.ClearDatabase()
	})
}

// Close the underlying DB usign the drivers.
func (l *DBLoader) Close(driver string) error {
	log.Info("Close database")

	drvr, err := database.From(driver)
	if err != nil {
		return err
	}

	return drvr.Close()
}

// SanityCheckBlockchain checks the head and the tail of the blockchain to avoid
// inconsistencies and a faulty bootstrap.
func (l *DBLoader) SanityCheckBlockchain(startAt, firstBlocksAmount uint64) error {
	var height uint64

	// Verify first N blocks
	err := l.db.View(func(t database.Transaction) error {
		h, err := t.FetchBlockHashByHeight(startAt)
		if err != nil {
			return err
		}

		prevHeader, err := t.FetchBlockHeader(h)
		if err != nil {
			return err
		}

		for height = startAt + 1; height <= firstBlocksAmount; height++ {
			hash, err := t.FetchBlockHashByHeight(height)

			if err == database.ErrBlockNotFound {
				// we reach the tip
				return nil
			}

			if err != nil {
				return err
			}

			header, err := t.FetchBlockHeader(hash)
			if err != nil {
				return err
			}

			if !bytes.Equal(header.PrevBlockHash, prevHeader.Hash) {
				return fmt.Errorf("invalid block hash at height %d", height)
			}

			prevHeader = header
		}

		return nil
	})
	if err != nil {
		return err
	}

	// TODO: Verify last blocks
	return nil
}

// LoadTip returns the tip of the chain.
func (l *DBLoader) LoadTip() (*block.Block, []byte, error) {
	var tip *block.Block
	var persistedHash []byte

	err := l.db.Update(func(t database.Transaction) error {
		s, err := t.FetchRegistry()
		if err != nil {
			// TODO: maybe log the error here and diversify between empty
			// results and actual errors

			// Store Genesis Block, if a modern node runs
			err = t.StoreBlock(l.genesis, true)
			if err != nil {
				return err
			}

			tip = l.genesis
			persistedHash = l.genesis.Header.Hash
			return nil
		}

		// Reconstruct chain tip
		h, err := t.FetchBlockHeader(s.TipHash)
		if err != nil {
			return err
		}

		txs, err := t.FetchBlockTxs(s.TipHash)
		if err != nil {
			return err
		}

		tip = &block.Block{Header: h, Txs: txs}
		persistedHash = s.PersistedHash

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	// Verify chain state. There shouldn't be any blocks higher than chainTip
	err = l.db.View(func(t database.Transaction) error {
		nextHeight := tip.Header.Height + 1

		hash, e := t.FetchBlockHashByHeight(nextHeight)
		// Check if error is nil and the hash is set
		if e == nil && len(hash) > 0 {
			err = fmt.Errorf("state points at %d height but the tip is higher", tip.Header.Height)
			log.WithError(err).Warn("loader failed")
		}
		// TODO: this throws ErrBlockNotFound in tests. Should we propagate the
		// error?
		//if e != nil {
		//	return e
		//}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return tip, persistedHash, nil
}
