package test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-wallet/block"
	"github.com/dusk-network/dusk-wallet/transactions"
	"github.com/syndtr/goleveldb/leveldb"

	// Import here any supported drivers to verify if they are fully compliant
	// to the blockchain database layer requirements
	"io/ioutil"
	"math"
	"os"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

var (
	// A Driver context
	drvr     database.Driver
	drvrName string
	db       database.DB
	storeDir string
	// Sample chain data to populate DB
	blocks []*block.Block
)

// TestMain defines minimum requirements (unit tests) that each Driver must
// satisfy.
//
// An additional role of TestMain (and the entire package) is to provide a
// simplified and working guideline of database driver usage.
//
// Note TestMain must clean up all resources on completion
func TestMain(m *testing.M) {

	var code int
	// Run on all registered drivers.
	for _, driverName := range database.Drivers() {

		code = _TestDriver(m, driverName)
		// the exit code might be needed on proper CI execution
		if code != 0 {
			os.Exit(code)
		}
	}
	os.Exit(code)
}

// _TestDriver executes all tests (declared in this file) in the context of a
// driver specified by driverName
func _TestDriver(m *testing.M, driverName string) int {

	// Cleanup TestMain iteration context
	defer func() {
		blocks = make([]*block.Block, 0)
		db = nil
		drvr = nil
		storeDir = ""
		atomic.StoreUint64(&heightCounter, 0)
	}()

	// Create a temp folder with random name
	var err error
	storeDir, err = ioutil.TempDir(os.TempDir(), driverName+"_temp_store_")
	if err != nil {
		fmt.Println(err)
		return 1
	}

	// Cleanup backend files only when running in testing mode
	defer os.RemoveAll(storeDir)

	// Retrieve a handler to an existing driver.
	drvr, err = database.From(driverName)
	drvrName = driverName

	if err != nil {
		fmt.Println(err)
		return 1
	}

	// Create a Database instance to use the temp directory. Multiple
	// database instances can work concurrently
	db, err = drvr.Open(storeDir, protocol.DevNet, false)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	defer db.Close()

	// Generate a few blocks to be used as sample objects
	// Less blocks are used as we have CI out-of-memory error
	t := &testing.T{}
	blocks, err = generateChainBlocks(t, 10)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	fmt.Println("Sample blocks generated")

	if len(blocks) == 0 {
		fmt.Println("Empty block slice. That's weird")
		return 1
	}

	// Try to store all blocks in append-only manner
	err = storeBlocks(nil, db, blocks)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	// Now run all tests which would use the provided context
	code := m.Run()

	if drvrName != lite.DriverName {
		code += _TestPersistence()
	}

	return code
}

func TestStoreBlock(test *testing.T) {

	// Generate additional blocks to store
	genBlocks, err := generateChainBlocks(test, 2)
	if err != nil {
		test.Fatal(err.Error())
	}

	done := false
	err = db.Update(func(t database.Transaction) error {
		for _, block := range genBlocks {
			err := t.StoreBlock(block)
			if err != nil {
				return err
			}
		}
		done = true
		return nil
	})

	if err != nil {
		test.Fatal(err.Error())
	}

	if !done {
		test.Fatal("No work done by the transaction")
	}

	// Ensure chain tip is updated too
	err = db.View(func(t database.Transaction) error {
		s, err := t.FetchState()
		if err != nil {
			return err
		}

		if !bytes.Equal(genBlocks[len(genBlocks)-1].Header.Hash, s.TipHash) {
			return fmt.Errorf("invalid chain tip")
		}
		return nil
	})

	if err != nil {
		test.Fatal(err.Error())
	}
}
func TestFetchBlockExists(test *testing.T) {

	test.Parallel()

	// Verify all blocks can be found by Header.Hash
	_ = db.View(func(t database.Transaction) error {
		for _, block := range blocks {
			exists, err := t.FetchBlockExists(block.Header.Hash)
			if err != nil {
				test.Fatalf(err.Error())
				return nil
			}

			if !exists {
				test.Fatalf("Block with Height %d was not found", block.Header.Height)
				return nil
			}
		}
		return nil
	})

	// FetchBlockExists with error
	_ = db.View(func(t database.Transaction) error {
		randHash, _ := crypto.RandEntropy(32)
		exists, err := t.FetchBlockExists(randHash)
		if err != database.ErrBlockNotFound {
			test.Fatalf("ErrBlockNotFound expected here when block with this height not found")
			return nil
		}

		if exists {
			test.Fatalf("Block not supposed to exist")
		}

		return nil
	})
}
func TestFetchBlockHeader(test *testing.T) {

	test.Parallel()

	// Verify all blocks headers can be fetched by Header.Hash
	err := db.View(func(t database.Transaction) error {
		for _, b := range blocks {
			fheader, err := t.FetchBlockHeader(b.Header.Hash)
			if err != nil {
				return err
			}

			// Get bytes of the fetched block.Header
			fetchedBuf := new(bytes.Buffer)
			_ = marshalling.MarshalHeader(fetchedBuf, fheader)

			// Get bytes of the origin block.Header
			originBuf := new(bytes.Buffer)
			_ = marshalling.MarshalHeader(originBuf, b.Header)

			// Ensure the fetched header is what the original header bytes are
			if !bytes.Equal(originBuf.Bytes(), fetchedBuf.Bytes()) {
				return errors.New("block.Header not retrieved properly from storage")
			}
		}
		return nil
	})

	if err != nil {
		test.Fatal(err.Error())
	}

	// Block lookup by hash with error
	_ = db.View(func(t database.Transaction) error {
		randHash, _ := crypto.RandEntropy(64)
		_, err := t.FetchBlockHeader(randHash)
		if err != database.ErrBlockNotFound {
			test.Fatalf("ErrBlockNotFound expected here when block with this height not found")
			return nil
		}

		return nil
	})
}
func TestFetchBlockTxs(test *testing.T) {

	test.Parallel()

	// Verify all blocks transactions can be fetched by Header.Hash
	err := db.View(func(t database.Transaction) error {
		for _, block := range blocks {

			// Fetch all transactions that belong to this block
			fblockTxs, err := t.FetchBlockTxs(block.Header.Hash)

			if err != nil {
				test.Fatalf(err.Error())
			}

			if len(block.Txs) != len(fblockTxs) {
				return errors.New("wrong number of fetched txs")
			}

			// Ensure all retrieved transactions are equal to the original Block.Txs
			// and the transactions order is the same too
			for index, oBlockTx := range block.Txs {

				if index >= len(fblockTxs) {
					return errors.New("missing instance of transactions.Transaction")
				}

				// Get bytes of the fetched transactions.Transaction
				fblockTx := fblockTxs[index]
				fetchedBuf := new(bytes.Buffer)
				_ = marshalling.MarshalTx(fetchedBuf, fblockTx)

				if len(fetchedBuf.Bytes()) == 0 {
					test.Fatal("Empty tx fetched")
				}

				// Get bytes of the origin transactions.Transaction to compare with
				originBuf := new(bytes.Buffer)
				_ = marshalling.MarshalTx(originBuf, oBlockTx)

				if !bytes.Equal(originBuf.Bytes(), fetchedBuf.Bytes()) {
					return errors.New("transactions.Transaction not retrieved properly from storage")
				}
			}
		}
		return nil
	})

	if err != nil {
		test.Fatal(err.Error())
	}
}
func TestFetchBlockHashByHeight(test *testing.T) {

	test.Parallel()

	// Blocks lookup by Height
	_ = db.View(func(t database.Transaction) error {
		for _, block := range blocks {
			headerHash, err := t.FetchBlockHashByHeight(block.Header.Height)
			if err != nil {
				test.Fatalf(err.Error())
				return nil
			}

			if !bytes.Equal(block.Header.Hash, headerHash) {
				test.Fatalf("FetchBlockHeaderByHeight failed on height %d", block.Header.Height)
				return nil
			}
		}
		return nil
	})

	// Blocks lookup by Height with error
	_ = db.View(func(t database.Transaction) error {
		heightEntropy, _ := crypto.RandEntropy(64)
		randHeight := binary.LittleEndian.Uint64(heightEntropy)

		_, err := t.FetchBlockHashByHeight(randHeight)
		if err != database.ErrBlockNotFound {
			test.Fatalf("ErrBlockNotFound expected here when block with this height not found")
			return nil
		}

		return nil
	})
}

func TestFetchKeyImageExists(test *testing.T) {

	test.Parallel()

	// Ensure all KeyImages have been stored to the KeyImage "table"
	err := db.View(func(t database.Transaction) error {
		for _, block := range blocks {
			for _, tx := range block.Txs {
				for _, input := range tx.StandardTx().Inputs {

					if len(input.KeyImage.Bytes()) == 0 {
						test.Fatal("Testing with empty keyImage")
					}

					exists, txID, err := t.FetchKeyImageExists(input.KeyImage.Bytes())

					if !exists {
						test.Fatal("FetchKeyImageExists cannot find keyImage")
					}

					if txID == nil {
						test.Fatal("FetchKeyImageExists found keyImage on invalid tx")
					}

					if err != nil {
						test.Fatal(err.Error())
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		test.Fatal(err.Error())
	}

	// Ensure it fails properly when a non-existing KeyImage is checked
	_ = db.View(func(t database.Transaction) error {
		invalidKeyImage, _ := crypto.RandEntropy(32)

		if len(invalidKeyImage) == 0 {
			test.Fatal("Testing with empty KeyImage")
		}

		exists, txID, err := t.FetchKeyImageExists(invalidKeyImage)

		if exists {
			test.Fatal("KeyImage is not supposed to be found")
		}

		if txID != nil {
			test.Fatal("Invalid TxID for non-existing KeyImage")
		}

		if err != database.ErrKeyImageNotFound {
			test.Fatal("ErrKeyImageNotFound is expected when fetching non-existing KeyImage")
		}
		return nil
	})
}

// TestAtomicUpdates ensures no change is applied into storage state when DB
// writable tx does fail
func TestAtomicUpdates(test *testing.T) {

	// This test ensures that the underlying storage state does not change.
	// That said, no parallelism should be applied.
	// test.Parallel()

	genBlocks := generateRandomBlocks(test, 2)

	// Save current storage state to compare later
	// Supported only in heavy.DB for now
	var snapshotBefore interface{}
	if drvr.Name() == heavy.DriverName {
		snapshotBefore, _ = db.(heavy.DB).GetSnapshot()
		defer snapshotBefore.(*leveldb.Snapshot).Release()
	}

	// Try to store all genBlocks and make it fail at last iteration of
	// read-write Tx
	forcedError := errors.New("force majeure situation")
	err := db.Update(func(t database.Transaction) error {

		for height, block := range genBlocks {
			err := t.StoreBlock(block)
			if err != nil {
				return err
			}

			if height == len(genBlocks)-1 {
				// Simulate an exception on storing last block. As a result we
				// expect to see no changes applied into backend storage state
				return forcedError
			}
		}
		return nil
	})

	if err != forcedError {
		test.Fatalf("ForcedError must be returned from previous statement. Failure is not always a bad thing")
	}

	// Check there are no changes applied into storage state by simply comparing
	// the leveldb snapshots
	//
	// Supported only in heavy.DB for now
	if drvr.Name() == heavy.DriverName {
		snapshotAfter, _ := db.(heavy.DB).GetSnapshot()
		defer snapshotAfter.Release()

		if snapshotAfter.String() != snapshotBefore.(*leveldb.Snapshot).String() {
			test.Fatalf("Backend storage state has changed.")
		}
	}
}

// TestReadOnlyTx ensures that a read-only DB tx cannot touch the storage state
func TestReadOnlyTx(test *testing.T) {

	test.Parallel()

	// Save current storage state to compare later
	// Supported only in heavy.DB for now
	var snapshotBefore interface{}
	if drvr.Name() == heavy.DriverName {
		snapshotBefore, _ = db.(heavy.DB).GetSnapshot()
		defer snapshotBefore.(*leveldb.Snapshot).Release()
	}

	// Try to call StoreBlock on a read-only DB Tx
	err := db.View(func(t database.Transaction) error {
		for _, block := range blocks {
			err := t.StoreBlock(block)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err == nil {
		test.Fatal("Any writable Tx must not be allowed on read-only DB Tx")
	}

	// Check there are no changes applied into storage state by simply comparing
	// the leveldb snapshots
	//
	// Supported only in heavy.DB for now
	if drvr.Name() == heavy.DriverName {
		snapshotAfter, _ := db.(heavy.DB).GetSnapshot()
		defer snapshotAfter.Release()

		if snapshotAfter.String() != snapshotBefore.(*leveldb.Snapshot).String() {
			test.Fatalf("Backend storage state has changed.")
		}
	}
}

// TestReadOnlyDB_Mode ensures a DB in read-only mode can only run read-only Tx
func TestReadOnlyDB_Mode(test *testing.T) {

	// Skip it for lite driver. Readonly DB mode will be reconsidered with
	// another issue
	if drvrName == lite.DriverName {
		test.Skip()
	}

	// Create database in read-write mode
	readonly := false
	dbReadWrite, err := drvr.Open(storeDir, protocol.DevNet, readonly)
	if err != nil {
		test.Fatal(err.Error())
	}

	if dbReadWrite == nil {
		test.Fatalf("Cannot create read-write database ")
	}

	defer dbReadWrite.Close()

	// Re-open the storage in read-only mode
	readonly = true
	dbReadOnly, err := drvr.Open(storeDir, protocol.DevNet, readonly)
	if err != nil {
		test.Fatal(err.Error())
	}

	if dbReadOnly == nil {
		test.Fatalf("Cannot open database in read-only mode")
	}

	defer dbReadOnly.Close()

	// Initialize db and a slice of 10 blocks with 2 transactions.Transaction each
	genBlocks, err := generateChainBlocks(test, 2)
	if err != nil {
		test.Fatal(err.Error())
	}

	// Store all blocks with read-write DB
	err = dbReadWrite.Update(func(t database.Transaction) error {
		for _, block := range genBlocks {
			err := t.StoreBlock(block)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		test.Fatal(err.Error())
	}

	// Storage Lookup by Height to ensure read-only DB can read
	_ = dbReadOnly.View(func(t database.Transaction) error {
		for _, block := range genBlocks {
			headerHash, err := t.FetchBlockHashByHeight(uint64(block.Header.Height))
			if err != nil {
				test.Fatalf(err.Error())
				return nil
			}

			if !bytes.Equal(block.Header.Hash, headerHash) {
				test.Fatalf("FetchBlockHeaderByHeight failed on height %d", block.Header.Height)
				return nil
			}
		}
		return nil
	})

	// Ensure read-only DB cannot write
	err = dbReadOnly.Update(func(t database.Transaction) error {
		for _, block := range blocks {
			err := t.StoreBlock(block)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err == nil {
		test.Fatal("A read-only DB is not permitted to make storage changes")
	}
}

func TestFetchBlockTxByHash(test *testing.T) {

	test.Parallel()

	var maxTxToFetch uint16 = 30

	done := false

	// Ensure we can fetch one by one each transaction by its TxID without
	// providing block.header.hash
	err := db.View(func(t database.Transaction) error {
		for _, block := range blocks {
			for txIndex, originTx := range block.Txs {

				// FetchBlockTxByHash
				txID, _ := originTx.CalculateHash()
				fetchedTx, fetchedIndex, _, err := t.FetchBlockTxByHash(txID)

				if err != nil {
					test.Fatal(err.Error())
				}

				/*
					if !bytes.Equal(fetchedBlockHash, block.Header.Hash) {
						test.Fatal("This tx does not belong to the right block")
					}
				*/

				if fetchedIndex != uint32(txIndex) {
					test.Fatal("Invalid index fetched")
				}

				fetchedBuf := new(bytes.Buffer)
				_ = marshalling.MarshalTx(fetchedBuf, fetchedTx)

				if len(fetchedBuf.Bytes()) == 0 {
					test.Fatal("Empty tx fetched")
				}

				originBuf := new(bytes.Buffer)
				_ = marshalling.MarshalTx(originBuf, originTx)

				if !bytes.Equal(fetchedBuf.Bytes(), originBuf.Bytes()) {
					test.Fatal("Invalid tx fetched")
				}

				done = true

				// Limit the overall number of tx to fetch
				maxTxToFetch--
				if maxTxToFetch == 0 {
					return nil
				}
			}
		}
		return nil
	})

	if err != nil {
		test.Fatal(err.Error())
	}

	if !done {
		test.Fatal("Incomplete test")
	}

	// Ensure it fails properly when txID is non-existing tx
	// FetchBlockTxByHash
	_ = db.View(func(t database.Transaction) error {
		invalidTxID, _ := crypto.RandEntropy(32)

		// FetchBlockTxByHash
		tx, txIndex, fetchedBlockHash, err := t.FetchBlockTxByHash(invalidTxID)

		if err != database.ErrTxNotFound {
			test.Fatal("ErrTxNotFound is expected when non-existing Tx is looked up")
		}

		if txIndex != math.MaxUint32 {
			test.Fatal("Invalid index fetched")
		}

		if tx != nil || fetchedBlockHash != nil {
			test.Fatal("Found non-existing tx?")
		}
		return nil
	})
}

func TestClearDatabase(test *testing.T) {
	err := db.Update(func(t database.Transaction) error {
		return t.ClearDatabase()
	})

	if err != nil {
		test.Fatal(err)
	}

	err = db.View(func(t database.Transaction) error {
		// All lookups should now fail
		for _, block := range blocks {
			blk, err := t.FetchBlock(block.Header.Hash)
			if err == nil && blk != nil {
				return errors.New("database was not empty")
			}
		}

		return nil
	})

	// repopulate db for the other tests
	if err := storeBlocks(test, db, blocks); err != nil {
		test.Fatal(err)
	}
}

func TestFetchOutputExists(test *testing.T) {
	test.Parallel()

	err := db.View(func(t database.Transaction) error {
		for _, block := range blocks {
			for _, tx := range block.Txs {
				for _, output := range tx.StandardTx().Outputs {
					exists, err := t.FetchOutputExists(output.PubKey.P.Bytes())
					if err != nil {
						return err
					}

					if !exists {
						test.Fatal("output key missing")
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		test.Fatal(err)
	}
}

func TestFetchOutputUnlockHeight(test *testing.T) {
	test.Parallel()

	err := db.View(func(t database.Transaction) error {
		for _, block := range blocks {
			for _, tx := range block.Txs {
				for i, output := range tx.StandardTx().Outputs {
					unlockHeight, err := t.FetchOutputUnlockHeight(output.PubKey.P.Bytes())
					if err != nil {
						return err
					}

					// If it's a bid or a stake, the unlockheight should
					// be non-zero for the first output
					if i == 0 && (tx.Type() == transactions.BidType || tx.Type() == transactions.StakeType) {
						if unlockHeight == 0 {
							test.Fatal("found bid or stake with 0 unlockheight")
						}
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		test.Fatal(err)
	}
}

func TestFetchDecoys(test *testing.T) {
	test.Parallel()

	// 7 is the standard number of decoys
	numDecoys := 7
	hits := 0
	err := db.View(func(t database.Transaction) error {
		decoys := t.FetchDecoys(numDecoys)
		// We should have at least 90 txs in our database, so plenty of
		// decoys to choose from, and under no circumstance should we
		// come up short.
		if len(decoys) != numDecoys {
			return errors.New("did not receive requested amount of decoys")
		}

		// Make sure these decoy points belong to transactions in
		// our db, which are unlocked.
		for _, decoy := range decoys {
			// Make sure it's a real output
			exists, err := t.FetchOutputExists(decoy.Bytes())
			if err != nil {
				return err
			}

			if !exists {
				return errors.New("fetched a decoy for which there is no output entry")
			}

			unlockHeight, err := t.FetchOutputUnlockHeight(decoy.Bytes())
			if err != nil {
				return err
			}

			if unlockHeight == 0 {
				hits++
			}
		}

		return nil
	})

	if err != nil {
		test.Fatal(err)
	}

	if hits != numDecoys {
		test.Fatalf("incorrect amount of hits - %v/%v", hits, numDecoys)
	}
}

// _TestPersistence tries to ensure if driver provides persistence storage.
// The procedure is simply based on:
// 1. Close the driver
// 2. Rename storage folder
// 3. Reload the driver
// This can be called only if all tests have completed.
// It returns result code 0 if all checks pass.
func _TestPersistence() int {

	// Closing the driver should release all allocated resources
	drvr.Close()

	// Rename the storage folder
	newStoreDir := storeDir + "_renamed"

	err := os.Rename(storeDir, newStoreDir)
	if err != nil {
		fmt.Printf("TestPersistence failed: %v\n", err.Error())
		return 1
	}

	storeDir = newStoreDir
	defer os.RemoveAll(storeDir)

	// Force reload storage and fetch blocks
	{
		db, err := drvr.Open(newStoreDir, protocol.DevNet, true)
		defer drvr.Close()

		// For instance, `resource temporarily unavailable` would be observed if
		// _storage.Close() is missed
		if err != nil {
			fmt.Printf("TestPersistence failed: %v\n", err)
			return 1
		}

		// Blocks lookup by Height
		err = db.View(func(t database.Transaction) error {
			for _, block := range blocks {
				headerHash, err := t.FetchBlockHashByHeight(block.Header.Height)
				if err != nil {
					return err
				}

				if !bytes.Equal(block.Header.Hash, headerHash) {
					return fmt.Errorf("couldn't fetch block on height %d", block.Header.Height)
				}
			}
			return nil
		})

		if err != nil {
			fmt.Printf("TestPersistence failed: %v\n", err.Error())
			return 1
		}

	}

	fmt.Printf("--- PASS: TestPersistence\n")

	return 0
}
