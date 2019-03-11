package database

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

// Driver is the interface that must be implemented by a database
// driver.
type Driver interface {
	// Open returns a new connection to the database.
	// The path is a string in a driver-specific format.
	Open(path string, network protocol.Magic, readonly bool) (DB, error)

	// Returns unique driver identifier to be registered with
	Name() string
}

// Tx represents transaction layer.
// A Database driver must provide a robust implementation of each method
// This is what is exposed/visible to all upper layers so changes here
// should be carefully considered
type Tx interface {

	// Read-write Transactions
	WriteHeader(h *block.Header) error

	// Read-only Transactions
	GetBlockHeaderByHash(hash []byte) (*block.Header, error)

	// Atomic storage.
	Commit() error
	Rollback() error
}

// DB a thin layer on top of block chain DB
type DB interface {

	// To provide a managed execution a read-only transaction
	View(fn func(tx Tx) error) error
	// To provide a managed execution of read-write transaction
	Update(fn func(tx Tx) error) error

	// TBD
	Close() error
}
