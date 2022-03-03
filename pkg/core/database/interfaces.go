// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package database

import (
	"errors"
	"math"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
)

var (
	// Common blockchain database errors. See also database/testing for the
	// cases where they are returned.

	// ErrKeyImageNotFound returned on a keyImage lookup.
	ErrKeyImageNotFound = errors.New("database: keyImage not found")
	// ErrTxNotFound returned on a tx lookup by hash.
	ErrTxNotFound = errors.New("database: transaction not found")
	// ErrBlockNotFound returned on a block lookup by hash or height.
	ErrBlockNotFound = errors.New("database: block not found")
	// ErrStateNotFound returned on missing state db entry.
	ErrStateNotFound = errors.New("database: state not found")
	// ErrOutputNotFound returned on output lookup during tx verification.
	ErrOutputNotFound = errors.New("database: output not found")

	// AnyTxType is used as a filter value on FetchBlockTxByHash.
	AnyTxType = transactions.TxType(math.MaxUint8)
)

// A Driver represents an application programming interface for accessing
// blockchain database management systems.
//
// It is conceptually similar to ODBC for DBMS.
type Driver interface {
	// Open returns a new connection to a blockchain database. The path is a
	// string in a driver-specific format.
	Open(path string, network protocol.Magic, readonly bool) (DB, error)

	// Close terminates all DB connections and closes underlying storage.
	Close() error

	// Name returns a unique identifier that can be used to register the driver.
	Name() string
}

// Transaction represents transaction layer. Transaction should provide basic
// transactions to fetch and store blockchain data.
//
// To simplify code reading with database pkg we should use 'Tx' to refer to
// blockchain transaction and 'Transaction' to refer to database transaction.
type Transaction interface {
	// Read-only transactions.

	FetchBlockHeader(hash []byte) (*block.Header, error)
	// Fetch all of the Txs that belong to a block with this header.hash.
	FetchBlockTxs(hash []byte) ([]transactions.ContractCall, error)
	// Fetch tx by txID. If succeeds, it returns tx data, tx index and
	// hash of the block it belongs to.
	FetchBlockTxByHash(txID []byte) (tx transactions.ContractCall, txIndex uint32, blockHeaderHash []byte, err error)
	FetchBlockHashByHeight(height uint64) ([]byte, error)
	FetchBlockExists(hash []byte) (bool, error)
	// Fetch chain registry (chain tip hash, persisted block etc).
	FetchRegistry() (*Registry, error)

	// Read-write transactions
	// Store the next chain block in a append-only manner
	// Overwrites only if block with same hash already stored
	// Not to be called concurrently, as it updates chain tip.
	StoreBlock(block *block.Block, persisted bool) error

	// FetchBlock will return a block, given a hash.
	FetchBlock(hash []byte) (*block.Block, error)

	// FetchCurrentHeight returns the height of the most recently stored
	// block in the database.
	FetchCurrentHeight() (uint64, error)

	// FetchBlockHeightSince try to find height of a block generated around
	// sinceUnixTime starting the search from height (tip - offset).
	FetchBlockHeightSince(sinceUnixTime int64, offset uint64) (uint64, error)

	// StoreCandidateMessage will...
	StoreCandidateMessage(cm block.Block) error

	FetchCandidateMessage(hash []byte) (block.Block, error)

	ClearCandidateMessages() error

	// ClearDatabase will remove all information from the database.
	ClearDatabase() error

	// Atomic storage.
	Commit() error
	Rollback() error
	Close()
}

// DB is a thin layer on top of Transaction providing a manageable execution.
type DB interface {
	// View provides a managed execution of a read-only transaction.
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

// Registry represents a set database records that provide chain metadata.
type Registry struct {
	TipHash       []byte
	PersistedHash []byte
}
