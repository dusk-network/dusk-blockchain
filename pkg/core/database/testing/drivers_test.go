package test

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/heavy"
	_ "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/lite"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"io/ioutil"
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
	for _, driverName := range database.Drivers() {
		code = _TestDriver(m, driverName)
		// the exit code might be needed by proper CI execution
		if code != 0 {
			os.Exit(code)
		}
	}
	os.Exit(code)
}

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

	// Retrieve a handler to an existing driver. If no errors, driver.Close
	// must be called when database is not used anymore.
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

	// Cleanup backend files only when running in testing mode
	defer os.RemoveAll(storeDir)

	// Generate a few blocks to be used as mock objects
	blocks, err = generateBlocks(1000, 10)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	// Try to store all blocks in concurrent manner for less than 10 seconds
	err = storeBlocksAsync(nil, db, blocks, time.Duration(10*time.Second))
	if err != nil {
		fmt.Println(err)
		return 1
	}

	// Now run all tests which would use the provided context
	code := m.Run()
	return code
}

func TestStoreBlock(t *testing.T) {

	t.Parallel()

	genBlocks, err := generateBlocks(10, 10)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = db.Update(func(tx database.Tx) error {
		for _, block := range genBlocks {
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
}
func TestFetchBlockExists(t *testing.T) {

	t.Parallel()

	// Verify all blocks have been stored successfully read-only Tx
	_ = db.View(func(tx database.Tx) error {
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
}
func TestFetchBlockHeader(t *testing.T) {

	t.Parallel()

	// Verify all blocks headers data have been stored successfully read-only Tx
	err := db.View(func(tx database.Tx) error {
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

	t.Parallel()

	// Verify all blocks Txs have been stored successfully read Tx
	err := db.View(func(tx database.Tx) error {
		for _, block := range blocks {

			// Fetch all transactions that belong to this block
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
func TestFetchBlockHashByHeight(t *testing.T) {

	t.Parallel()

	// Storage Lookup by Height
	_ = db.View(func(tx database.Tx) error {
		for _, block := range blocks {
			headerHash, err := tx.FetchBlockHashByHeight(block.Header.Height)
			if err != nil {
				t.Fatalf(err.Error())
				return nil
			}

			if !bytes.Equal(block.Header.Hash, headerHash) {
				t.Fatalf("FetchBlockHeaderByHeight failed on height %d", block.Header.Height)
				return nil
			}
		}
		return nil
	})
}

// TestAtomicUpdates ensures no change is applied into storage state when DB
// writable tx does fail
func TestAtomicUpdates(t *testing.T) {

	t.Parallel()

	genBlocks, err := generateBlocks(10, 2)

	if err != nil {
		t.Fatal(err.Error())
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
	err = db.Update(func(tx database.Tx) error {

		for height, block := range genBlocks {
			err := tx.StoreBlock(block)
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
		t.Fatalf("ForcedError must be returned from previous statement. Failure is not always a bad thing")
	}

	// Check there are no changes applied into storage state by simply comparing
	// the leveldb snapshots
	//
	// Supported only in heavy.DB for now
	if drvr.Name() == heavy.DriverName {
		snapshotAfter, _ := db.(heavy.DB).GetSnapshot()
		defer snapshotAfter.Release()

		if snapshotAfter.String() != snapshotBefore.(*leveldb.Snapshot).String() {
			t.Fatalf("Backend storage state has changed.")
		}
	}
}

// TestReadOnlyTx ensures that a read-only DB tx cannot touch the storage state
func TestReadOnlyTx(t *testing.T) {

	t.Parallel()

	// Save current storage state to compare later
	// Supported only in heavy.DB for now
	var snapshotBefore interface{}
	if drvr.Name() == heavy.DriverName {
		snapshotBefore, _ = db.(heavy.DB).GetSnapshot()
		defer snapshotBefore.(*leveldb.Snapshot).Release()
	}

	// Try to call StoreBlock on a read-only DB Tx
	err := db.View(func(tx database.Tx) error {
		for _, block := range blocks {
			err := tx.StoreBlock(block)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err == nil {
		t.Fatal("Any writable Tx must not be allowed on read-only DB Tx")
	}

	// Check there are no changes applied into storage state by simply comparing
	// the leveldb snapshots
	//
	// Supported only in heavy.DB for now
	if drvr.Name() == heavy.DriverName {
		snapshotAfter, _ := db.(heavy.DB).GetSnapshot()
		defer snapshotAfter.Release()

		if snapshotAfter.String() != snapshotBefore.(*leveldb.Snapshot).String() {
			t.Fatalf("Backend storage state has changed.")
		}
	}
}
func TestReadOnlyDB_Mode(t *testing.T) {

	t.Parallel()

	// Create database in read-write mode
	readonly := false
	dbReadWrite, err := drvr.Open(storeDir, protocol.DevNet, readonly)
	if err != nil {
		t.Fatal(err.Error())
	}

	if dbReadWrite == nil {
		t.Fatalf("Cannot create read-write database ")
	}

	defer dbReadWrite.Close()

	// Re-open the storage in read-only mode
	readonly = true
	dbReadOnly, err := drvr.Open(storeDir, protocol.DevNet, readonly)
	if err != nil {
		t.Fatal(err.Error())
	}

	if dbReadOnly == nil {
		t.Fatalf("Cannot open database in read-only mode")
	}

	defer dbReadOnly.Close()

	// Initialize db and a slice of 100 blocks with 2 transactions.Stealth each
	genBlocks, err := generateBlocks(100, 2)
	if err != nil {
		t.Fatal(err.Error())
	}

	// Store all blocks with read-write DB
	err = dbReadWrite.Update(func(tx database.Tx) error {
		for _, block := range genBlocks {
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

	// Storage Lookup by Height to ensure read-only DB can read
	_ = dbReadOnly.View(func(tx database.Tx) error {
		for _, block := range genBlocks {
			headerHash, err := tx.FetchBlockHashByHeight(uint64(block.Header.Height))
			if err != nil {
				t.Fatalf(err.Error())
				return nil
			}

			if !bytes.Equal(block.Header.Hash, headerHash) {
				t.Fatalf("FetchBlockHeaderByHeight failed on height %d", block.Header.Height)
				return nil
			}
		}
		return nil
	})

	// Ensure read-only DB cannot write
	err = dbReadOnly.Update(func(tx database.Tx) error {
		for _, block := range blocks {
			err := tx.StoreBlock(block)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err == nil {
		t.Fatal("A read-only DB is not permitted to make storage changes")
	}
}
