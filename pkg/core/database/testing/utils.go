package test

import (
	"errors"
	"fmt"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var (
	// heightCounter is used by generateBlocks as atomic counter
	heightCounter uint64

	// sampleTxsBatchCount
	// Number of txs per a sample block is  = ( 1 + sampleTxsBatchCount * 4)
	sampleTxsBatchCount uint16 = 2
)

// storeBlocksAsync is a helper function to store a slice of blocks in a
// concurrent manner.
func storeBlocksAsync(test *testing.T, db database.DB, blocks []*block.Block, timeoutDuration time.Duration) error {

	routinesCount := runtime.NumCPU()
	blocksCount := len(blocks)
	var wg sync.WaitGroup
	// For each slice of N blocks build a batch to be performed concurrently
	for batchIndex := 0; batchIndex <= blocksCount/routinesCount; batchIndex++ {

		// get a slice of all blocks
		from := routinesCount * batchIndex
		to := from + routinesCount

		if to > blocksCount {
			// half-open interval reslicing
			to = blocksCount
		}

		// Start a separate unit to perform a DB tx with multiple StoreBlock
		// calls
		wg.Add(1)
		go func(blocks []*block.Block, wg *sync.WaitGroup) {

			defer wg.Done()
			_ = db.Update(func(t database.Transaction) error {

				for _, block := range blocks {
					err := t.StoreBlock(block)
					if err != nil {
						fmt.Print(err.Error())
						return err
					}
				}
				return nil
			})

		}(blocks[from:to], &wg)
	}

	// Wait here for all updates to complete or just timeout in case of a
	// deadlock.
	timeouted := waitTimeout(&wg, timeoutDuration)

	if timeouted {
		// Also it might be due to too slow write call
		return errors.New("Seems like we've got a deadlock situation on storing blocks in concurrent way")
	}

	return nil
}

// A helper function to generate a set of blocks as mock objects
func generateBlocks(test *testing.T, blocksCount int) ([]*block.Block, error) {

	overallBlockTxs := (1 + 4*int(sampleTxsBatchCount))
	overallTxsCount := blocksCount * overallBlockTxs
	fmt.Printf("--- MSG  Generate sample data of %d blocks with %d txs each (overall txs %d)\n", blocksCount, overallBlockTxs, overallTxsCount)

	newBlocks := make([]*block.Block, blocksCount)
	for i := 0; i < blocksCount; i++ {
		b := helper.RandomBlock(test, atomic.AddUint64(&heightCounter, 1), sampleTxsBatchCount)
		newBlocks[i] = b
	}

	// Make all txs calculate and cache hash value.
	for _, b := range newBlocks {
		for _, tx := range b.Txs {
			_, err := tx.CalculateHash()

			if err != nil {
				return nil, err
			}
		}
	}

	return newBlocks, nil
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}
