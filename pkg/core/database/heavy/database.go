package heavy

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"os"
	"sync"
)

var (
	// See openStorage for detailed explanation
	_storage   *leveldb.DB
	_storageMu sync.Mutex
)

// DB on top of underlying storage syndtr/goleveldb/leveldb
type DB struct {
	// an alias to the global storage var
	storage *leveldb.DB

	// Read-only mode provided at heavy.DB level If true, accepts read-only Tx
	readOnly bool
}

// openStorage is a wrapper around leveldb.OpenFile to provide singleton
// leveldb.DB instance
//
// leveldb.OpenFile returns a new filesystem-backed storage implementation with
// the given path. This also acquire a file lock, so any subsequent attempt to
// open the same path will fail.
//
// Even opening with leveldb.Options{ ReadOnly: true} an err EAGAIN is returned
func openStorage(path string) (*leveldb.DB, error) {
	_storageMu.Lock()
	defer _storageMu.Unlock()

	var err error
	if _storage == nil {
		var s *leveldb.DB
		s, err = leveldb.OpenFile(path, nil)

		// Try to recover if corrupted
		if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
			s, err = leveldb.RecoverFile(path, nil)
		}

		if _, accessdenied := err.(*os.PathError); accessdenied {
			err = errors.New("Could not open or create db")
		}

		if s != nil && err == nil {
			_storage = s
		}
	}
	return _storage, err
}

// NewDatabase create or open backend storage (goleveldb) on the specified path
// Readonly option is pseudo read-only mode implemented by heavy.Database
func NewDatabase(path string, network protocol.Magic, readonly bool) (database.DB, error) {

	storage, err := openStorage(path)
	if err != nil {
		return nil, err
	}

	return DB{storage, readonly}, nil
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
	tx := &Tx{writable: writable,
		db:       &db,
		snapshot: snapshot,
		batch:    batch,
		closed:   false}

	return tx, nil
}

// Update provides an execution of managed, read-write Tx
func (db DB) Update(fn func(database.Tx) error) error {

	// Create a writable tx for atomic update
	tx, err := db.Begin(true)
	if err != nil {
		return err
	}

	defer tx.Close()

	// If an error is returned from the function then rollback and return error.
	err = fn(tx)

	// Handing a panic event to rollback tx is not needed. If we fail at that
	// point, no commit will be applied into backend storage

	if err != nil {
		return err
	}

	return tx.Commit()
}

// View provides an execution of managed, read-only Tx
func (db DB) View(fn func(database.Tx) error) error {

	tx, err := db.Begin(false)
	if err != nil {
		return err
	}

	defer tx.Close()

	// If an error is returned from the function then pass it through.
	err = fn(tx)
	return err
}

func (db DB) isOpen() bool {
	// Unfortunately, goleveldb does not expose DB.IsOpen/DB.IsClose calls
	return db.storage != nil
}

// Close does not close the underlying storage as we need to reuse it within
// another DB instances. Note that we rely on the Finalizer that goleveldb is
// setting at the point of leveldb.DB.openDB
func (db DB) Close() error {
	db.storage = nil
	return nil
}

func (db DB) LogStats() {
	var stat leveldb.DBStats
	err := db.storage.Stats(&stat)
	if err != nil {
		fmt.Print(err)
	}
	fmt.Printf("%+v", stat)
}

func (db DB) GetSnapshot() (*leveldb.Snapshot, error) {
	return db.storage.GetSnapshot()
}
