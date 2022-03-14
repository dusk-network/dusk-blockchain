// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package lite

import (
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
)

type (
	key   [64]byte
	table map[key][]byte
	memdb [maxInd]table
)

const (
	// Block table index.
	blocksInd = iota
	txsInd
	txHashInd
	heightInd
	stateInd
	candidateInd
	persistedInd
	maxInd
)

var stateKey = []byte{1}

// DB represents the db struct.
type DB struct {
	storage  memdb
	mu       sync.RWMutex
	readOnly bool
	path     string
}

// NewDatabase returns a DB instance.
// This should be the ideal situation with lowest latency on storing or fetching data.
// In-memory only (as result autoDeleted).
// multi-instances (no singleton).
func NewDatabase(path string, readonly bool) (database.DB, error) {
	var db *DB
	var tables [maxInd]table

	for i := 0; i < len(tables); i++ {
		tables[i] = make(table)
	}

	db = &DB{path: path, readOnly: readonly, storage: tables}

	return db, nil
}

// Begin builds read-only or read-write Transaction.
func (db *DB) Begin(writable bool) (database.Transaction, error) {
	var batch memdb

	if writable && !db.readOnly {
		for i := range batch {
			batch[i] = make(table)
		}
	}

	t := &transaction{
		writable: writable,
		db:       db, batch: batch,
	}

	return t, nil
}

// Update the DB within a transaction.
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

// View performs the equivalent of a Select SQL statement.
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

// Close is actually a dummy method on a lite driver.
func (db *DB) Close() error {
	return nil
}
