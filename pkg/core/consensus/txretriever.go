package consensus

import (
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
)

// TxRetriever is a simple searcher, who's responsibility is to find consensus-related transactions, when given
// an identifier, and a comparison function.
type TxRetriever struct {
	db          database.DB
	compareFunc func([]transactions.Transaction, []byte) (transactions.Transaction, error)
}

func NewTxRetriever(db database.DB, compareFunc func([]transactions.Transaction, []byte) (transactions.Transaction, error)) *TxRetriever {
	// Get a db connection, if none was given.
	if db == nil {
		_, db = heavy.CreateDBConnection()
	}

	return &TxRetriever{
		db:          db,
		compareFunc: compareFunc,
	}
}

// SearchForTx will walk up the blockchain to find a transaction with the corresponding `identifier`,
// by running the `compareFunc` on each tx for every found block. If the `compareFunc` finds a match,
// then this tx is returned. This tx can then be used by a consensus component to start or reset itself.
func (i *TxRetriever) SearchForTx(identifier []byte) (transactions.Transaction, error) {
	currentHeight := i.getCurrentHeight()
	searchingHeight := i.getSearchingHeight(currentHeight)

	for {
		blk, err := i.getBlock(searchingHeight)
		if err != nil {
			break
		}

		tx, err := i.compareFunc(blk.Txs, identifier)
		if err != nil {
			searchingHeight++
			continue
		}

		return tx, nil
	}

	return nil, errors.New("could not find corresponding value for specified item")
}

// Consensus transactions can only be valid for the maximum locktime. So, we will begin our search at
// the tip height, minus the maximum locktime.
func (i *TxRetriever) getSearchingHeight(currentHeight uint64) uint64 {
	if currentHeight < transactions.MaxLockTime {
		return 0
	}
	return currentHeight - transactions.MaxLockTime
}

func (i *TxRetriever) getBlock(searchingHeight uint64) (*block.Block, error) {
	var b *block.Block
	err := i.db.View(func(t database.Transaction) error {
		hash, err := t.FetchBlockHashByHeight(searchingHeight)
		if err != nil {
			return err
		}

		b, err = t.FetchBlock(hash)
		return err
	})

	return b, err
}

func (i *TxRetriever) getCurrentHeight() (currentHeight uint64) {
	err := i.db.View(func(t database.Transaction) error {
		var err error
		currentHeight, err = t.FetchCurrentHeight()
		return err
	})

	if err != nil {
		// If we can not get the current height, it means we have no DB state object, or the header of the tip hash is missing
		// Neither should ever happen, so we panic
		panic(err)
	}

	return
}
