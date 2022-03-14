// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package heavy

import (
	"os"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
)

var (
	// See openStorage for detailed explanation.
	_storage   *leveldb.DB
	_storageMu sync.Mutex
)

// DB on top of underlying storage syndtr/goleveldb/leveldb.
type DB struct {
	// an alias to the global storage var.
	storage *leveldb.DB

	// Read-only mode provided at heavy.DB level. If true, accepts read-only Transaction.
	readOnly bool
}

// openStorage is a wrapper around leveldb.OpenFile to provide singleton
// leveldb.DB instance.
//
// leveldb.OpenFile returns a new filesystem-backed storage implementation with
// the given path. This also acquire a file lock, so any subsequent attempt to
// open the same path will fail.
//
// Even opening with leveldb.Options{ ReadOnly: true} an err EAGAIN is returned.
func openStorage(path string) (*leveldb.DB, error) {
	_storageMu.Lock()
	defer _storageMu.Unlock()

	var err error

	if _storage == nil {
		var s *leveldb.DB
		s, err = leveldb.OpenFile(path, nil)

		// Try to recover if corrupted.
		if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
			s, err = leveldb.RecoverFile(path, nil)
		}

		if _, accessdenied := err.(*os.PathError); accessdenied {
			err = errors.New("could not open or create db")
		}

		if s != nil && err == nil {
			_storage = s
		}
	}
	return _storage, err
}

// closeStorage should safely close the underlying storage.
func closeStorage() error {
	_storageMu.Lock()
	defer _storageMu.Unlock()

	if _storage != nil {
		err := _storage.Close()
		_storage = nil
		return err
	}

	return errors.New("invalid storage")
}

// NewDatabase create or open backend storage (goleveldb) located at the
// specified path. Readonly option is pseudo read-only mode implemented by
// heavy.Database. Not to be confused with read-only goleveldb mode.
func NewDatabase(path string, readonly bool) (database.DB, error) {
	storage, err := openStorage(path)
	if err != nil {
		return nil, err
	}

	return DB{storage, readonly}, nil
}

// Begin builds read-only or read-write Transaction.
func (db DB) Begin(writable bool) (database.Transaction, error) {
	// If the database was opened with DB.readonly flag true, we cannot create
	// a writable transaction
	if db.readOnly && writable {
		return nil, errors.New("database is read-only")
	}

	// In case of writable transaction, it's now the time to obtain writer lock.
	// This is not necessary in case of LevelDB store due its concurrency
	// Control as describe below

	// A database may only be opened by one process at a time. The leveldb
	// implementation acquires a lock from the operating system to prevent
	// misuse. Within a single process, the same leveldb::DB object may be
	// safely shared by multiple concurrent threads. I.e., different threads may
	// write into or fetch iterators or call Get on the same database without
	// any external synchronization

	// Exit if the database is not open yet.
	if !db.isOpen() {
		return nil, errors.New("database is not open")
	}

	// Snapshot is a DB snapshot. It must be released on Transaction.Close()
	snapshot, err := db.storage.GetSnapshot()
	if err != nil {
		return nil, err
	}

	// Batch to be used by a writable Transaction.
	var batch *leveldb.Batch
	if writable {
		batch = new(leveldb.Batch)
	}

	// Create a transaction instance. Mind Transaction.Close() must be called
	// when Transaction is done
	t := &transaction{
		writable: writable,
		db:       &db,
		snapshot: snapshot,
		batch:    batch,
		closed:   false,
	}

	return t, nil
}

// Update a record within a transaction.
func (db DB) Update(fn func(database.Transaction) error) error {
	// Create a writable transaction for atomic update.
	t, err := db.Begin(true)
	if err != nil {
		return err
	}

	defer t.Close()

	// If an error is returned from the function then rollback and return error.
	// rollback here simply means to skip the commit step
	err = fn(t)

	// Handing a panic event to rollback Transaction is not needed. If we fail
	// at that point, no commit will be applied into backend storage

	if err != nil {
		return err
	}

	return t.Commit()
}

// View is the equivalent of a Select SQL statement.
func (db DB) View(fn func(database.Transaction) error) error {
	t, err := db.Begin(false)
	if err != nil {
		return err
	}

	defer t.Close()
	return fn(t)
}

func (db DB) isOpen() bool {
	// Unfortunately, goleveldb does not expose DB.IsOpen/DB.IsClose calls
	return db.storage != nil
}

// Close does not close the underlying storage as we need to reuse it within
// another DB instances. Note that we rely on the Finalizer that goleveldb is
// setting at the point of leveldb.DB.openDB to call leveldb.DB.Close() when
// the process is terminating.
func (db DB) Close() error {
	db.storage = nil
	return nil
}

// GetSnapshot returns current storage snapshot. To be used only by
// database/testing pkg.
func (db DB) GetSnapshot() (*leveldb.Snapshot, error) {
	return db.storage.GetSnapshot()
}
