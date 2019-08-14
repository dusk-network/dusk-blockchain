package initiation

import (
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
)

// Initiator is a simple searcher which takes in a byte item, and a callback to run comparisons with this item,
// against block transactions. It's purpose is to find a corresponding value given an item, and return it to the caller.
type Initiator struct {
	db          database.DB
	item        []byte
	compareFunc func([]transactions.Transaction, []byte) ([]byte, error)
}

func NewInitiator(db database.DB, item []byte, compareFunc func([]transactions.Transaction, []byte) ([]byte, error)) *Initiator {
	// Get a db connection, if none was passed.
	if db == nil {
		_, db = heavy.CreateDBConnection()
	}

	return &Initiator{
		db:          db,
		item:        item,
		compareFunc: compareFunc,
	}
}

// SearchForValue will walk up the blockchain to find a transaction with the corresponding `item`,
// by running the `compareFunc` on each tx for every found block. If the item matches, a corresponding value is returned.
// This value can then be used by a consensus component to start or reset itself.
func (i *Initiator) SearchForValue() ([]byte, error) {
	currentHeight := i.getCurrentHeight()
	searchingHeight := i.getSearchingHeight(currentHeight)

	for {
		blk, err := i.getBlock(searchingHeight)
		if err != nil {
			break
		}

		value, err := i.compareFunc(blk.Txs, i.item)
		if err != nil {
			searchingHeight++
			continue
		}

		return value, nil
	}

	return nil, errors.New("could not find corresponding value for specified item")
}

// Consensus transactions can only be valid for the maximum locktime. So, we will begin our search at
// the tip height, minus the maximum locktime.
func (i *Initiator) getSearchingHeight(currentHeight uint64) uint64 {
	if currentHeight < transactions.MaxLockTime {
		return 0
	}
	return currentHeight - transactions.MaxLockTime
}

func (i *Initiator) getBlock(searchingHeight uint64) (*block.Block, error) {
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

func (i *Initiator) getCurrentHeight() (currentHeight uint64) {
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
