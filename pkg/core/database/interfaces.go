package database

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

// Driver is the interface that must be implemented by a database
// driver.
type Driver interface {
	// Open returns a new connection to the database.
	// The name is a string in a driver-specific format.
	Open(name string, network protocol.Magic, readonly bool) (DB, error)
}

// Tx represents transaction layer.
// A Database driver must provide a robust implementation of each method
// This is what is exposed/visible to all upper layers so changes here
// should be carefully considered
type Tx interface {

	// Read-write Transactions

	WriteHeader(h *block.Header) error
	//WriteBlockTransactions(blocks []*block.Block) error

	// Read-only Transactions

	//GetBlockHeaderByHeight(height uint64) (*block.Header, error)
	//GetBlockHeaderByRange(start uint64, stop uint64) ([]*block.Header, error)
	//GetBlockHeaderByHash(hash []byte) (*block.Header, error)
	//GetLatestHeader returns the most recent header.
	GetBlockHeaderByHash(hash []byte) (*block.Header, error)

	// GetBlock will return the block from the received hash
	//GetBlock(hash []byte) (*block.Block, error)
	//BlockExists(hdr *block.Header) bool

	// Atomic storage.
	Commit() error
	Rollback() error
}

// DB a thin layer on top of block chain DB
type DB interface {
	Type() string
	View(fn func(tx Tx) error) error
	Update(fn func(tx Tx) error) error
	Close() error
}
