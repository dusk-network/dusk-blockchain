package heavy

import (
	"bytes"
	"errors"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"
)

func TestFetchBlockExists(t *testing.T) {

	db, blocks, err := newTestContext(t, 30, 1)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer db.Close()
	defer os.RemoveAll(db.path)

	// Store all blocks
	// read-write Tx
	err = db.Update(func(tx database.Tx) error {
		for _, block := range blocks {
			err := tx.StoreBlock(block)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		t.Fatal(err.Error())
	}

	// Verify all blocks have been stored successfully
	// read-only Tx
	err = db.View(func(tx database.Tx) error {
		for _, block := range blocks {
			exists, err := tx.FetchBlockExists(block.Header.Hash)
			if err != nil {
				t.Fatalf(err.Error())
				return nil
			}

			if !exists {
				t.Fatalf("Block with Height %d was not found", block.Header.Height)
				return nil
			}
		}
		return nil
	})

	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestFetchBlockHeader(t *testing.T) {

	// Timeout threshold not to be exceeded when running this test
	timeoutDuration := time.Duration(10 * time.Second)
	db, blocks, err := newTestContext(t, 88, 10)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer db.Close()
	defer os.RemoveAll(db.path)

	// Store all blocks
	err = storeBlocksAsync(t, db, blocks, timeoutDuration)

	if err != nil {
		t.Fatal(err.Error())
	}

	// Verify all blocks headers data have been stored successfully
	// read-only Tx
	err = db.View(func(tx database.Tx) error {
		for _, block := range blocks {
			fheader, err := tx.FetchBlockHeader(block.Header.Hash)
			if err != nil {
				return err
			}

			// Get bytes of the fetched block.Header
			fetchedBuf := new(bytes.Buffer)
			_ = fheader.Encode(fetchedBuf)

			// Get bytes of the origin block.Header
			originBuf := new(bytes.Buffer)
			_ = block.Header.Encode(originBuf)

			if !bytes.Equal(originBuf.Bytes(), fetchedBuf.Bytes()) {
				return errors.New("block.Header not retrieved properly from storage")
			}
		}
		return nil
	})

	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestFetchTransactions(t *testing.T) {

	// Timeout threshold not to be exceeded when running this test
	// Currently not more than ~0.2s are needed to complete the storing
	timeoutDuration := time.Duration(10 * time.Second)
	db, blocks, err := newTestContext(t, 88, 10)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer db.Close()
	defer os.RemoveAll(db.path)

	// Store all blocks
	err = storeBlocksAsync(t, db, blocks, timeoutDuration)

	if err != nil {
		t.Fatal(err.Error())
	}

	// Verify all blocks Txs have been stored successfully
	// read Tx
	err = db.View(func(tx database.Tx) error {
		for _, block := range blocks {

			// Fetch all transactions that belongs to this block
			fblockTxs, err := tx.FetchBlockTransactions(block.Header.Hash)

			if err != nil {
				t.Fatalf(err.Error())
			}

			// Ensure all retrieved transactions are equal to origin Block.Txs
			// and the transactions order is kept
			for index, v := range block.Txs {

				if index >= len(fblockTxs) {
					return errors.New("Missing instances of transactions.Stealth")
				}

				// Get bytes of the fetched transactions.Stealth
				fblockTx := fblockTxs[index].(*transactions.Stealth)
				fetchedBuf := new(bytes.Buffer)
				_ = fblockTx.Encode(fetchedBuf)

				// Get bytes of the origin transactions.Stealth to compare with
				oBlockTx := v.(*transactions.Stealth)
				originBuf := new(bytes.Buffer)
				_ = oBlockTx.Encode(originBuf)

				if !bytes.Equal(originBuf.Bytes(), fetchedBuf.Bytes()) {
					return errors.New("transactions.Stealth not retrieved properly from storage")
				}
			}
		}
		return nil
	})

	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestFetchByHeight(t *testing.T) {

	db, blocks, err := newTestContext(t, 100, 1)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer db.Close()
	defer os.RemoveAll(db.path)

	// Store all blocks
	err = db.Update(func(tx database.Tx) error {
		for _, block := range blocks {
			err := tx.StoreBlock(block)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		t.Fatal(err.Error())
	}

	// Storage Lookup by Height
	_ = db.View(func(tx database.Tx) error {
		for height, block := range blocks {
			headerHash, err := tx.FetchBlockHashByHeight(uint64(height))
			if err != nil {
				t.Fatalf(err.Error())
			}

			if !bytes.Equal(block.Header.Hash, headerHash) {
				t.Fatalf("FetchBlockHeaderByHeight failed on height %d", height)
				return nil
			}
		}
		return nil
	})
}

// TestAtomicUpdates ensures no change is applied into storage state when
// DB writable tx does fail
func TestAtomicUpdates(t *testing.T) {

	blocksCount := 10
	db, blocks, err := newTestContext(t, blocksCount, 2)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer db.Close()
	defer os.RemoveAll(db.path)

	snapshotBefore, _ := db.storage.GetSnapshot()
	defer snapshotBefore.Release()

	// Try to store all blocks and make it fail at last iteration
	// read-write Tx
	forcedError := errors.New("force majeure situation")
	err = db.Update(func(tx database.Tx) error {

		for height, block := range blocks {
			err := tx.StoreBlock(block)
			if err != nil {
				return err
			}

			if height == blocksCount-1 {
				// Simulate an exception on storing last block
				// As a result we expect to see no changes applied
				// into backend storage state
				return forcedError
			}
		}
		return nil
	})

	if err != forcedError {
		t.Fatalf("We expect to see here the so-called ForcedError. Failure is not always a bad thing")
	}

	// Check there are no changes applied into storage state by
	// simply comparing the leveldb snapshots
	snapshotAfter, _ := db.storage.GetSnapshot()
	defer snapshotAfter.Release()

	if snapshotAfter.String() != snapshotBefore.String() {
		t.Fatalf("Backend storage state has changed.")
	}
}

// TestReadOnlyTx ensures that a read-only DB tx cannot touch the
// storage state
func TestReadOnlyTx(t *testing.T) {

	db, blocks, err := newTestContext(t, 3, 1)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer db.Close()
	defer os.RemoveAll(db.path)

	snapshotBefore, _ := db.storage.GetSnapshot()
	defer snapshotBefore.Release()

	// Try to call StoreBlock when on read-only DB Tx
	err = db.View(func(tx database.Tx) error {
		for _, block := range blocks {
			err := tx.StoreBlock(block)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err == nil {
		t.Fatal("StoreBlock should not be allowed on read-only DB Tx")
	}

	// Check there are no changes applied into storage state by
	// simply comparing the leveldb snapshots
	snapshotAfter, _ := db.storage.GetSnapshot()
	defer snapshotAfter.Release()

	if snapshotAfter.String() != snapshotBefore.String() {
		t.Fatalf("Backend storage state has changed.")
	}
}

// Helper functions used above

// helperStoreBlocksAsync is a helper function to store a set of blocks in a concurrent manner
// Make a DB transaction with a few StoreBlock calls
func storeBlocksAsync(t *testing.T, db *DB, blocks []*block.Block, timeoutDuration time.Duration) error {

	batchBlocksCount := 10
	blocksCount := len(blocks)
	var wg sync.WaitGroup
	// For each slice of N blocks built a batch to be performed concurrently
	for batchIndex := 0; batchIndex <= blocksCount/batchBlocksCount; batchIndex++ {

		// get a slice of all blocks
		from := batchBlocksCount * batchIndex
		to := from + batchBlocksCount

		if to > blocksCount {
			// half-open interval reslicing
			to = blocksCount
		}

		// Start a separate unit to perform a DB tx with multiple StoreBlock calls
		wg.Add(1)
		go func(blocks []*block.Block, wg *sync.WaitGroup) {

			defer wg.Done()
			_ = db.Update(func(tx database.Tx) error {

				for _, block := range blocks {
					err := tx.StoreBlock(block)
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
		// Also it might be due to too many
		return errors.New("Seems like we've got a deadlock situation on storing blocks in concurrent way")
	}

	return nil
}

// newTestContext must guarantee that each test run is based on
// brand new (isolated) DB and a slice of block.Block and no collisions can happen between tests
//
// TODO: A TestContext struct can encapsulate db.Close and dbPath deletion
func newTestContext(t *testing.T, blocksCount int, txsCount int) (*DB, []*block.Block, error) {

	if t != nil {
		t.Logf("TestContext on %d block with %d tx each (overall %d txs)", blocksCount, txsCount, txsCount*blocksCount)
	}

	// Create a temp folder with random name
	storeDir, err := ioutil.TempDir(os.TempDir(), "leveldb_store")
	if err != nil {
		t.Fatal(err.Error())
	}

	// Create a Database instance to use the temp directory
	db, err := NewDatabase(storeDir, false)

	if err != nil {
		if t != nil {
			t.Fatal(err.Error())
		}
		os.RemoveAll(storeDir)
		return nil, nil, err
	}

	// Generate a few blocks to be used as mock objects
	blocks, err := generateBlocks(blocksCount, txsCount)

	return db, blocks, err
}

// A helper function to generate a set of blocks as mock objects
func generateBlocks(blocksCount int, txsCount int) ([]*block.Block, error) {

	blocks := make([]*block.Block, blocksCount)

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
		b.Header.Height = uint64(index)

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

		blocks[index] = b
	}

	return blocks, nil
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
