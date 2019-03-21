package database

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

// Driver is the interface that must be implemented by a database
// driver.
type Driver interface {
	// Open returns a new connection to a blockchain database. The path is a
	// string in a driver-specific format.
	Open(path string, network protocol.Magic, readonly bool) (DB, error)

	// Returns unique driver identifier to be registered with
	Name() string
}

// Tx represents transaction layer.
// A Database driver must provide a robust implementation of each method
// This is what is exposed/visible to all upper layers so changes here
// should be carefully considered
//
// Tx should provide basic transactions to fetch and store blockchain data.
// High-level transactions are supposed to be implemented by the Consumer
type Tx interface {

	// Read-only Transactions
	FetchBlockHeader(hash []byte) (*block.Header, error)
	FetchBlockTransactions(hash []byte) ([]merkletree.Payload, error)
	FetchBlockHashByHeight(height uint64) ([]byte, error)
	FetchBlockExists(hash []byte) (bool, error)

	// Read-write Transactions
	StoreBlock(block *block.Block) error

	// Atomic storage.
	Commit() error
	Rollback() error
	Close()
}

// DB a thin layer on top of block chain DB
type DB interface {

	// To provide a managed execution of a read-only transaction
	View(fn func(tx Tx) error) error
	// To provide a managed execution of a read-write transaction
	Update(fn func(tx Tx) error) error

	Close() error
}
