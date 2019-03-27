package database

import (
	"errors"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

var (
	// Common blockchain database errors. See also database/testing for the
	// cases where they are returned

	// ErrTxNotFound returned on a keyImage lookup
	ErrKeyImageNotFound = errors.New("database: keyImage not found")
	// ErrTxNotFound returned on a tx lookup by hash
	ErrTxNotFound = errors.New("database: transaction not found")
	// ErrBlockNotFound returned on a block lookup by hash or height
	ErrBlockNotFound = errors.New("database: block not found")
)

// A Driver represents an application programming interface for accessing
// blockchain database management systems.
//
// It is conceptually similar to ODBC for DBMS
type Driver interface {
	// Open returns a new connection to a blockchain database. The path is a
	// string in a driver-specific format.
	Open(path string, network protocol.Magic, readonly bool) (DB, error)

	// Name returns a unique identifier that can be used to register the driver
	Name() string
}

// Transaction represents transaction layer. Transaction should provide basic
// transactions to fetch and store blockchain data.
//
// To simplify code reading with database pkg we should use 'Tx' to refer to
// blockchain transaction and 'Transaction' to refer to database transaction
type Transaction interface {

	// Read-only transactions

	FetchBlockHeader(hash []byte) (*block.Header, error)
	// Fetch all of the Txs that belong to a block with this header.hash
	FetchBlockTxs(hash []byte) ([]merkletree.Payload, error)
	// Fetch tx by txID. If succeeds, it returns tx data, tx index and
	// hash of the block it belongs to.
	FetchBlockTxByHash(txID []byte) (tx merkletree.Payload, txIndex uint32, blockHeaderHash []byte, err error)
	FetchBlockHashByHeight(height uint64) ([]byte, error)
	FetchBlockExists(hash []byte) (bool, error)

	// Check if an input keyImage is already stored. If succeeds, it returns
	// also txID the input belongs to
	FetchKeyImageExists(keyImage []byte) (exists bool, txID []byte, err error)

	// Read-write transactions
	StoreBlock(block *block.Block) error

	// Atomic storage
	Commit() error
	Rollback() error
	Close()
}

// DB is a thin layer on top of Transaction providing a manageable execution
type DB interface {

	// View provides a managed execution of a read-only transaction
	View(fn func(t Transaction) error) error

	// Update provides a managed execution of a read-write atomic transaction.
	//
	// An atomic transaction is an indivisible and irreducible series of
	// database operations such that either all occur, or nothing occurs.
	//
	// Transaction commit will happen only if no error is returned by `fn`
	// and no panic is raised on `fn` execution.
	Update(fn func(t Transaction) error) error

	Close() error
}
