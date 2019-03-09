package heavy

import (
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"os"
	"time"
)

var (
	driverName    = "heavy_v0.1.0"
	storageIsOpen = false
)

// DB on top of underlying storage syndtr/goleveldb/leveldb
// Note that this repo seems to be unofficial LevelDB porting to golang
// (not built by LevelDB Authors)
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
	// TODO: Support read-only LevelDB option
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

// Begin builds (read-only or read-write) Tx, do initial validations
func (db *DB) Begin(writable bool) (database.Tx, error) {
	// If the database was opened with Options.ReadOnly, return an error.
	if db.readOnly && writable {
		return nil, errors.New("Database is read-only")
	}

	// In case of writable Tx, it's now the time to obtain writer lock.
	// This is not necessary in case of LevelDB store due its concurrency Control as
	// describe below

	// A database may only be opened by one process at a time.
	// The leveldb implementation acquires a lock from the operating system to prevent misuse. Within a single process, the same leveldb::DB object may be safely shared by multiple concurrent threads. I.e., different threads may write into or fetch iterators or call Get on the same database without any external synchronization

	// Exit if the database is not open yet.
	if !db.isOpen() {
		return nil, errors.New("Database is not open")
	}

	// Create a transaction associated with the database.
	t := &Tx{writable: writable, db: db}

	return t, nil
}

// Update provides an execution of managed, read-write Tx
func (db *DB) Update(fn func(database.Tx) error) error {
	start := time.Now()
	t, err := db.Begin(true)
	if err != nil {
		return err
	}

	// If an error is returned from the function then rollback and return error.
	err = fn(t)
	duration := time.Since(start)
	log.WithField("prefix", "database").Debugf("Transaction duration %d", duration.Nanoseconds())
	return err
}

// View provides an execution of managed, read-only Tx
func (db *DB) View(fn func(database.Tx) error) error {
	start := time.Now()
	t, err := db.Begin(false)
	if err != nil {
		return err
	}

	// If an error is returned from the function then pass it through.
	err = fn(t)

	duration := time.Since(start)
	log.WithField("prefix", "database").Debugf("Transaction duration %d", duration.Nanoseconds())

	return nil
}

func (db *DB) isOpen() bool {
	return storageIsOpen
}

func (db *DB) Type() string {
	return driverName
}

func (db *DB) Close() error {
	return nil
}

func init() {
	storageIsOpen = false
}
