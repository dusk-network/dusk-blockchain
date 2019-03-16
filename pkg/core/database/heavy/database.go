package heavy

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"os"
)

var (
	storageIsOpen = false
)

// DB on top of underlying storage syndtr/goleveldb/leveldb
type DB struct {
	storage *leveldb.DB
	path    string

	// If true, accepts read-only Tx
	readOnly bool
}

// NewDatabase a singleton connection to storage
func NewDatabase(path string, readonly bool) (*DB, error) {

	if storageIsOpen {
		// Not recommended to be open more than once
		return nil, errors.New("Please note that LevelDB is not designed for multi-instance access")
	}

	// Open Dusk blockchain db or create (if it not alresdy exists)
	storage, err := leveldb.OpenFile(path, nil)

	// Try to recover if corrupted
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		storage, err = leveldb.RecoverFile(path, nil)
	}

	if _, accessdenied := err.(*os.PathError); accessdenied {
		return nil, errors.New("Could not open or create db")
	}

	storageIsOpen = true

	return &DB{storage, path, readonly}, nil
}

// Begin builds read-only or read-write Tx
func (db DB) Begin(writable bool) (database.Tx, error) {
	// If the database was opened with Options.ReadOnly, return an error.
	if db.readOnly && writable {
		return nil, errors.New("Database is read-only")
	}

	// In case of writable Tx, it's now the time to obtain writer lock. This is
	// not necessary in case of LevelDB store due its concurrency Control as
	// describe below

	// A database may only be opened by one process at a time. The leveldb
	// implementation acquires a lock from the operating system to prevent
	// misuse. Within a single process, the same leveldb::DB object may be
	// safely shared by multiple concurrent threads. I.e., different threads may
	// write into or fetch iterators or call Get on the same database without
	// any external synchronization

	// Exit if the database is not open yet.
	if !db.isOpen() {
		return nil, errors.New("Database is not open")
	}

	// Snapshot to be released on Tx.Close
	snapshot, err := db.storage.GetSnapshot()

	if err != nil {
		return nil, err
	}

	// Batch to be used by a writable Tx.
	var batch *leveldb.Batch = nil
	if writable {
		batch = new(leveldb.Batch)
	}

	// Create a transaction instance. Mind Tx.Close() must be called when Tx is
	// done
	t := &Tx{writable: writable,
		db:       &db,
		snapshot: snapshot,
		batch:    batch,
		closed:   false}

	return t, nil
}

// Update provides an execution of managed, read-write Tx
func (db DB) Update(fn func(database.Tx) error) error {

	// Create a writable tx for atomic update
	t, err := db.Begin(true)

	if err != nil {
		return err
	}

	defer t.Close()

	// If an error is returned from the function then rollback and return error.
	err = fn(t)

	// Handing a panic event to rollback tx is not needed. If we fail at that
	// point, no commit will be applied into backend storage

	if err != nil {
		return err
	}

	return t.Commit()
}

// View provides an execution of managed, read-only Tx
func (db DB) View(fn func(database.Tx) error) error {

	t, err := db.Begin(false)

	if err != nil {
		return err
	}

	defer t.Close()

	// If an error is returned from the function then pass it through.
	err = fn(t)

	return err
}

func (db DB) isOpen() bool {
	return storageIsOpen
}

func (db DB) Close() error {
	if storageIsOpen {
		db.storage.Close()
		storageIsOpen = false
	}
	return nil
}

func init() {
	storageIsOpen = false
}
