// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package test

import (
	"fmt"
	"sync/atomic"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
)

var (
	// heightCounter is used by generateBlocks as atomic counter.
	heightCounter uint64

	// sampleTxsBatchCount
	// Number of txs per a sample block is  = ( 1 + sampleTxsBatchCount * 4).
	sampleTxsBatchCount uint16 = 2
)

// storeBlocks is a helper function to store a slice of blocks in a
// sequential manner.
func storeBlocks(db database.DB, blocks []*block.Block) error {
	err := db.Update(func(t database.Transaction) error {
		for _, block := range blocks {
			// Store block
			err := t.StoreBlock(block, false)
			if err != nil {
				fmt.Print(err.Error())
				return err
			}
		}

		return nil
	})

	return err
}

// A helper function to generate a set of blocks that can be chained.
func generateChainBlocks(blocksCount int) ([]*block.Block, error) {
	overallBlockTxs := 1 + 4*int(sampleTxsBatchCount)
	overallTxsCount := blocksCount * overallBlockTxs
	newBlocks := make([]*block.Block, blocksCount)

	fmt.Printf("--- MSG  Generate sample data of %d blocks with %d txs each (overall txs %d)\n", blocksCount, overallBlockTxs, overallTxsCount)

	for i := 0; i < blocksCount; i++ {
		b := helper.RandomBlock(atomic.AddUint64(&heightCounter, 1), sampleTxsBatchCount)
		// assume consensus time is 10sec
		b.Header.Timestamp = int64(10 * b.Header.Height)

		for _, tx := range b.Txs {
			if _, err := tx.CalculateHash(); err != nil {
				return nil, err
			}
		}

		newBlocks[i] = b
	}

	return newBlocks, nil
}

func generateRandomBlocks(blocksCount int) []*block.Block {
	newBlocks := make([]*block.Block, blocksCount)
	for i := 0; i < blocksCount; i++ {
		newBlocks[i] = helper.RandomBlock(uint64(i+1), sampleTxsBatchCount)
	}
	return newBlocks
}
