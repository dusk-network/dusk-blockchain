package chain2

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
	// calling Loader.PerformSanityCheck
	SanityCheckHeight uint64 = 10
)

// DBLoader performs database prefetching and sanityChecks at node startup
type DBLoader struct {
	db database.DB

	// Unsure if the genesis block needs to be here
	genesis *block.Block

	// Output prefetched data
	chainTip *block.Block
}

// TODO: Temp DB()
func (l *DBLoader) DB() database.DB {
	return l.db
}

// SanityCheckBlock will verify whether we have not seed the block before
// (duplicate), perform a check on the block header and verifies the coinbase
// transactions. It leaves the bulk of transaction verification to the executor
// Return nil if the sanity check passes
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

	if err := verifiers.CheckMultiCoinbases(blk.Txs); err != nil {
		return err
	}

	return nil
}

// NewDBLoader returns a Loader which gets the Chain Tip from the DB
func NewDBLoader(db database.DB, genesis *block.Block) *DBLoader {
	return &DBLoader{db: db, genesis: genesis}
}

// Height returns the height of the blockchain stored in the DB
func (l *DBLoader) Height() (uint64, error) {
	var height uint64
	err := l.db.View(func(t database.Transaction) error {
		var err error
		height, err = t.FetchCurrentHeight()
		return err
	})
	return height, err
}

// Append stores a block in the DB
func (l *DBLoader) Append(blk *block.Block) error {
	return l.db.Update(func(t database.Transaction) error {
		return t.StoreBlock(blk)
	})
}

// BlockAt returns the block stored at a given height
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

// Clear the underlying DB
func (l *DBLoader) Clear() error {
	return l.db.Update(func(t database.Transaction) error {
		return t.ClearDatabase()
	})
}

// Close the underlying DB usign the drivers
func (l *DBLoader) Close(driver string) error {
	log.Info("Close database")
	drvr, err := database.From(driver)
	if err != nil {
		return err
	}

	return drvr.Close()
}

// PerformSanityCheck checks the head and the tail of the blockchain to avoid
// inconsistencies and a faulty bootstrap
func (l *DBLoader) PerformSanityCheck(startAt, firstBlocksAmount, lastBlocksAmount uint64) error {
	var height uint64
	var prevBlock *block.Block

	if startAt > 0 {
		return errors.New("performing sanity checks from arbitrary points is not supported yet")
	}

	if startAt == 0 {
		prevBlock = l.genesis
	}

	prevHeader := prevBlock.Header
	// Verify first N blocks
	err := l.db.View(func(t database.Transaction) error {

		// This will most likely verify genesis, unless the startAt parameter
		// is set to some other height. In case of genesis, failure here would mostly occur if mainnet node
		// loads testnet blockchain
		hash, err := t.FetchBlockHashByHeight(startAt)
		if err != nil {
			return err
		}

		if !bytes.Equal(prevHeader.Hash, hash) {
			return fmt.Errorf("invalid genesis block")
		}

		for height = 1; height <= firstBlocksAmount; height++ {
			hash, err := t.FetchBlockHashByHeight(height)

			if err == database.ErrBlockNotFound {
				// seems we reach the tip
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

	//TODO: Verify lastBlockAmount blocks
	return nil
}

// LoadTip returns the tip of the chain
func (l *DBLoader) LoadTip() (*block.Block, error) {
	var tip *block.Block
	err := l.db.Update(func(t database.Transaction) error {

		s, err := t.FetchState()
		if err != nil {
			// TODO: maybe log the error here and diversify between empty
			// results and actual errors

			// Store Genesis Block, if a modern node runs
			err := t.StoreBlock(l.genesis)
			if err != nil {
				return err
			}
			tip = l.genesis

		} else {
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
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Verify chain state. There shouldn't be any blocks higher than chainTip
	err = l.db.View(func(t database.Transaction) error {
		nextHeight := tip.Header.Height + 1
		hash, e := t.FetchBlockHashByHeight(nextHeight)
		// Check if error is nil and the hash is set
		if e == nil && len(hash) > 0 {
			return fmt.Errorf("state points at %d height but the tip is higher", tip.Header.Height)
		}
		// TODO: this throws ErrBlockNotFound in tests. Should we propagate the
		// error?
		//if e != nil {
		//	return e
		//}
		return nil
	})

	if err != nil {
		return nil, err
	}

	l.chainTip = tip
	return tip, nil
}
