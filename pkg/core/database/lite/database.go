package lite

import (
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

type key [64]byte
type table map[key][]byte
type memdb [maxInd]table

const (

	// block table index
	blocksInd = iota
	txsInd
	keyImagesInd
	candidatesTableInd
	heightInd
	stateInd
	maxInd
)

var (
	stateKey = []byte{1}
)

type DB struct {
	storage  memdb
	mu       sync.RWMutex
	readOnly bool
	path     string
}

var (
	pool map[string]*DB
	mu   sync.RWMutex
)

// This should be the ideal situation with lowest latency on storing or fetching data
// In-memory only (as result autoDeleted)
// multi-instances (no singlton)
func NewDatabase(path string, network protocol.Magic, readonly bool) (database.DB, error) {

	mu.Lock()
	defer mu.Unlock()

	if pool == nil {
		pool = make(map[string]*DB)
	}

	exists := false
	var db *DB
	if db, exists = pool[path]; !exists {

		var tables [maxInd]table
		for i := 0; i < len(tables); i++ {
			tables[i] = make(table)
		}

		db = &DB{path: path, readOnly: readonly, storage: tables}
		pool[path] = db
	}

	return db, nil
}

// Begin builds read-only or read-write Transaction
func (db *DB) Begin(writable bool) (database.Transaction, error) {

	var batch memdb
	if writable && !db.readOnly {
		for i := range batch {
			batch[i] = make(table)
		}
	}

	t := &transaction{writable: writable,
		db: db, batch: batch}

	return t, nil
}

func (db *DB) Update(fn func(database.Transaction) error) error {

	db.mu.Lock()
	defer db.mu.Unlock()

	// Create a writable transaction for atomic update
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

func (db *DB) View(fn func(database.Transaction) error) error {

	db.mu.RLock()
	defer db.mu.RUnlock()

	t, err := db.Begin(false)
	if err != nil {
		return err
	}

	defer t.Close()
	return fn(t)
}

func (db *DB) Close() error {
	mu.Lock()
	defer mu.Unlock()
	delete(pool, db.path)

	return nil
}
