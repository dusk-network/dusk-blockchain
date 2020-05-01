package test

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
)

var (
	// heightCounter is used by generateBlocks as atomic counter
	heightCounter uint64

	// sampleTxsBatchCount
	// Number of txs per a sample block is  = ( 1 + sampleTxsBatchCount * 4)
	sampleTxsBatchCount uint16 = 2
)

// storeBlocks is a helper function to store a slice of blocks in a
// sequential manner.
func storeBlocks(db database.DB, blocks []*block.Block) error {
	err := db.Update(func(t database.Transaction) error {
		for _, block := range blocks {
			// Store block
			err := t.StoreBlock(block)
			if err != nil {
				fmt.Print(err.Error())
				return err
			}
		}
		return nil
	})

	return err
}

// A helper function to generate a set of blocks that can be chained
func generateChainBlocks(test *testing.T, blocksCount int) ([]*block.Block, error) {

	overallBlockTxs := 1 + 4*int(sampleTxsBatchCount)
	overallTxsCount := blocksCount * overallBlockTxs
	fmt.Printf("--- MSG  Generate sample data of %d blocks with %d txs each (overall txs %d)\n", blocksCount, overallBlockTxs, overallTxsCount)

	newBlocks := make([]*block.Block, blocksCount)
	for i := 0; i < blocksCount; i++ {
		b := helper.RandomBlock(test, atomic.AddUint64(&heightCounter, 1), sampleTxsBatchCount)
		// assume consensus time is 10sec
		b.Header.Timestamp = int64(10 * b.Header.Height)

		for _, tx := range b.Txs {
			payload, err := block.NewSHA3Payload(tx)
			if err != nil {
				return nil, err
			}

			_, err = payload.CalculateHash()
			if err != nil {
				return nil, err
			}
		}

		newBlocks[i] = b
	}

	return newBlocks, nil
}

func generateRandomBlocks(test *testing.T, blocksCount int) []*block.Block {
	newBlocks := make([]*block.Block, blocksCount)
	for i := 0; i < blocksCount; i++ {
		newBlocks[i] = helper.RandomBlock(test, uint64(i+1), sampleTxsBatchCount)
	}
	return newBlocks
}
