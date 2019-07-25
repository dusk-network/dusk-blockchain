package chain

import (
	"bytes"
	"fmt"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
)

const (
	sanityCheckHeight uint64 = 10
)

// loader performs database prefetching and sanityChecks at node startup
type loader struct {
	db database.DB

	// Output prefetched data
	chainTip *block.Block
}

func newLoader(db database.DB) (*loader, error) {

	l := &loader{db: db}

	// Prefetch data from database that is needed by the node in advance
	if err := l.prefetch(); err != nil {
		return nil, err
	}

	// Perform database sanity check to ensure the database state is rational
	// before boostrapping all node subsystems
	if err := l.sanityCheck(); err != nil {
		return nil, err
	}

	return l, nil
}

func (l *loader) prefetch() error {

	if err := l.prefetchChainTip(); err != nil {
		return err
	}

	return nil
}

func (l *loader) sanityCheck() error {

	var height uint64
	// Verify first N blocks
	prevBlock := cfg.DecodeGenesis()
	prevHeader := prevBlock.Header
	err := l.db.View(func(t database.Transaction) error {

		// Verify genesis. Failure here would mostly occur if mainnet node
		// loads testnet blockchain
		hash, err := t.FetchBlockHashByHeight(0)
		if err != nil {
			return err
		}

		if !bytes.Equal(prevHeader.Hash, hash) {
			return fmt.Errorf("invalid genesis block")
		}

		for height = 1; height <= sanityCheckHeight; height++ {
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

	//TODO: Verify blocks latest N blocks

	return nil
}

func (l *loader) prefetchChainTip() error {

	err := l.db.Update(func(t database.Transaction) error {

		s, err := t.FetchState()
		if err != nil {

			// Store Genesis Block, if a modern node runs
			b := cfg.DecodeGenesis()
			err := t.StoreBlock(b)
			if err != nil {
				return err
			}
			l.chainTip = b
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

			l.chainTip = &block.Block{Header: h, Txs: txs}
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Verify chain state. There shouldn't be any blocks higher than chainTip
	err = l.db.View(func(t database.Transaction) error {
		nextHeight := l.chainTip.Header.Height + 1
		hash, err := t.FetchBlockHashByHeight(nextHeight)
		// Check if
		if err == nil && len(hash) > 0 {
			return fmt.Errorf("state points at %d height but the tip is higher", l.chainTip.Header.Height)
		}
		return nil
	})

	return err
}
