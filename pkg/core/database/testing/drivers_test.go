package test

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/heavy"
	_ "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/lite"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"io/ioutil"
	"math"
	"os"
	"testing"
	"time"
)

var (
	// A Driver context
	drvr     database.Driver
	db       database.DB
	storeDir string
	// Dummy data to populate DB initially
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
	blocks, err = generateBlocks(300, 10)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	if len(blocks) == 0 {
		fmt.Println("Empty block slice. That's weird")
		return 1
	}

	// Try to store all blocks in concurrent manner for less than 10 seconds
	err = storeBlocksAsync(nil, db, blocks, time.Duration(10*time.Second))
	if err != nil {
		fmt.Println(err)
		return 1
	}

	// Now run all tests which would use the provided context
	return m.Run()
}

func TestStoreBlock(test *testing.T) {

	test.Parallel()

	// Generate additional blocks to store
	genBlocks, err := generateBlocks(10, 10)
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
}
func TestFetchBlockHeader(test *testing.T) {

	test.Parallel()

	// Verify all blocks headers can be fetched by Header.Hash
	err := db.View(func(t database.Transaction) error {
		for _, block := range blocks {
			fheader, err := t.FetchBlockHeader(block.Header.Hash)
			if err != nil {
				return err
			}

			// Get bytes of the fetched block.Header
			fetchedBuf := new(bytes.Buffer)
			_ = fheader.Encode(fetchedBuf)

			// Get bytes of the origin block.Header
			originBuf := new(bytes.Buffer)
			_ = block.Header.Encode(originBuf)

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

			// Ensure all retrieved transactions are equal to the original Block.Txs
			// and the transactions order is the same too
			for index, v := range block.Txs {

				if index >= len(fblockTxs) {
					return errors.New("Missing instances of transactions.Stealth")
				}

				// Get bytes of the fetched transactions.Stealth
				fblockTx := fblockTxs[index].(*transactions.Stealth)
				fetchedBuf := new(bytes.Buffer)
				_ = fblockTx.Encode(fetchedBuf)

				if len(fetchedBuf.Bytes()) == 0 {
					test.Fatal("Empty tx fetched")
				}

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
}

func TestFetchKeyImageExists(test *testing.T) {

	test.Parallel()

	// Ensure all KeyImages have been stored to the KeyImage "table"
	err := db.View(func(t database.Transaction) error {
		for _, block := range blocks {
			for _, v := range block.Txs {
				tx := v.(*transactions.Stealth)

				var inputs []*transactions.Input
				switch tx.Type {
				case transactions.StandardType:
					typeInfo := tx.TypeInfo.(*transactions.Standard)
					inputs = typeInfo.Inputs

				case transactions.TimelockType:
					typeInfo := tx.TypeInfo.(*transactions.Standard)
					inputs = typeInfo.Inputs
				}

				for _, input := range inputs {

					if len(input.KeyImage) == 0 {
						test.Fatal("Testing with empty keyImage")
					}

					exists, txID, err := t.FetchKeyImageExists(input.KeyImage)

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

		if err == nil {
			test.Fatal("Missing error when fetching non-existing KeyImage")
		}
		return nil
	})
}

// TestAtomicUpdates ensures no change is applied into storage state when DB
// writable tx does fail
func TestAtomicUpdates(test *testing.T) {

	test.Parallel()

	genBlocks, err := generateBlocks(10, 2)

	if err != nil {
		test.Fatal(err.Error())
	}

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
	err = db.Update(func(t database.Transaction) error {

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

	test.Parallel()

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

	// Initialize db and a slice of 100 blocks with 2 transactions.Stealth each
	genBlocks, err := generateBlocks(100, 2)
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

	done := false
	// Ensure we can fetch one by one each transaction by its TxID without
	// providing block.header.hash
	err := db.View(func(t database.Transaction) error {
		for _, block := range blocks {
			for index, v := range block.Txs {
				originTx := v.(*transactions.Stealth)

				// FetchBlockTxByHash
				tx, fetchedIndex, fetchedBlockHash, err := t.FetchBlockTxByHash(originTx.R)

				if err != nil {
					test.Fatal(err.Error())
				}

				fetchedTx := tx.(*transactions.Stealth)

				if fetchedIndex != uint32(index) {
					test.Fatal("Invalid index fetched")
				}

				if !bytes.Equal(fetchedBlockHash, block.Header.Hash) {
					test.Fatal("This tx does not belong to the right block")
				}

				fetchedBuf := new(bytes.Buffer)
				_ = fetchedTx.Encode(fetchedBuf)

				if len(fetchedBuf.Bytes()) == 0 {
					test.Fatal("Empty tx fetched")
				}

				originBuf := new(bytes.Buffer)
				_ = originTx.Encode(originBuf)

				if !bytes.Equal(fetchedBuf.Bytes(), originBuf.Bytes()) {
					test.Fatal("Invalid tx fetched")
				}

				done = true
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
		tx, fetchedIndex, fetchedBlockHash, err := t.FetchBlockTxByHash(invalidTxID)

		if err == nil {
			test.Fatal("Error is expected when non-existing Tx is looked up")
		}

		if fetchedIndex != math.MaxUint32 {
			test.Fatal("Invalid index fetched")
		}

		if tx != nil || fetchedBlockHash != nil {
			test.Fatal("Found non-existing tx?")
		}
		return nil
	})
}
