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
}

// Tx represents transaction layer.
// A Database driver must provide a robust implementation of each method
// This is what is exposed/visible to all upper layers so changes here
// should be carefully considered
type Tx interface {

	// TODO: Define transactions to satisfy Dusk Block chain DB specifics
	// As convention any read-transaction can be prefixed with Get,
	// any write-transaction can be prefixed with Write

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

	// Returns unique database Type identifier
	Type() string

	// To provide a managed execution a read-only transaction
	View(fn func(tx Tx) error) error
	// To provide a managed execution of read-write transaction
	Update(fn func(tx Tx) error) error

	// TBD
	Close() error
}
