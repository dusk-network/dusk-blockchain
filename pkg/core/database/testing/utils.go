package test

import (
	"encoding/binary"
	"errors"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
	"runtime"
	"sync"
	"testing"
	"time"
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
func generateBlocks(blocksCount int, txsCount int) ([]*block.Block, error) {

	newBlocks := make([]*block.Block, blocksCount)

	for index := 0; index < blocksCount; index++ {
		b := block.NewBlock()

		// Add 10 transactions
		for i := 0; i < txsCount; i++ {
			byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

			sig, _ := crypto.RandEntropy(2000)

			txPubKey, _ := crypto.RandEntropy(32)
			pl := transactions.NewStandard(100)
			s := transactions.NewTX(transactions.StandardType, pl)
			in := transactions.NewInput(txPubKey, txPubKey, 0, sig)
			pl.AddInput(in)
			s.R = txPubKey

			out := transactions.NewOutput(200, byte32, sig)
			pl.AddOutput(out)
			if err := s.SetHash(); err != nil {
				return nil, err
			}

			b.AddTx(s)
		}

		// Set Height
		heightEntropy, _ := crypto.RandEntropy(64)
		b.Header.Height = binary.LittleEndian.Uint64(heightEntropy)

		// Spoof previous hash and seed
		h, _ := crypto.RandEntropy(32)
		b.Header.PrevBlock = h

		s, _ := crypto.RandEntropy(33)
		b.Header.Seed = s

		// Add cert image
		rand1, _ := crypto.RandEntropy(32)
		rand2, _ := crypto.RandEntropy(32)

		sig, _ := crypto.RandEntropy(33)

		slice := make([][]byte, 0)
		slice = append(slice, rand1)
		slice = append(slice, rand2)

		cert := &block.Certificate{
			BRBatchedSig: sig,
			BRStep:       4,
			BRPubKeys:    slice,
			SRBatchedSig: sig,
			SRStep:       2,
			SRPubKeys:    slice,
		}

		if err := cert.SetHash(); err != nil {
			return nil, err
		}

		if err := b.AddCertHash(cert); err != nil {
			return nil, err
		}

		// Finish off
		if err := b.SetRoot(); err != nil {
			return nil, err
		}

		b.Header.Timestamp = time.Now().Unix()
		if err := b.SetHash(); err != nil {
			return nil, err
		}

		newBlocks[index] = b
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
