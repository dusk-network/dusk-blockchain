package database

import (
	"errors"
	"math"

	"github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

var (
	// Common blockchain database errors. See also database/testing for the
	// cases where they are returned

	// ErrKeyImageNotFound returned on a keyImage lookup
	ErrKeyImageNotFound = errors.New("database: keyImage not found")
	// ErrTxNotFound returned on a tx lookup by hash
	ErrTxNotFound = errors.New("database: transaction not found")
	// ErrBlockNotFound returned on a block lookup by hash or height
	ErrBlockNotFound = errors.New("database: block not found")
	// ErrStateNotFound returned on missing state db entry
	ErrStateNotFound = errors.New("database: state not found")

	// AnyTxType is used as a filter value on FetchBlockTxByHash
	AnyTxType = transactions.TxType(math.MaxUint8)
)

// A Driver represents an application programming interface for accessing
// blockchain database management systems.
//
// It is conceptually similar to ODBC for DBMS
type Driver interface {
	// Open returns a new connection to a blockchain database. The path is a
	// string in a driver-specific format.
	Open(path string, network protocol.Magic, readonly bool) (DB, error)

	// Close terminates all DB connections and closes underlying storage
	Close() error

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
	FetchBlockTxs(hash []byte) ([]transactions.Transaction, error)
	// Fetch tx by txID. If succeeds, it returns tx data, tx index and
	// hash of the block it belongs to.
	FetchBlockTxByHash(txID []byte) (tx transactions.Transaction, txIndex uint32, blockHeaderHash []byte, err error)
	FetchBlockHashByHeight(height uint64) ([]byte, error)
	FetchBlockExists(hash []byte) (bool, error)
	// Fetch chain state information (chain tip hash)
	FetchState() (*State, error)

	// Check if an input keyImage is already stored. If succeeds, it returns
	// also txID the input belongs to
	FetchKeyImageExists(keyImage []byte) (exists bool, txID []byte, err error)

	// Fetch a candidate block by hash
	FetchCandidateBlock(hash []byte) (*block.Block, error)

	// Read-write transactions
	// Store the next chain block in a append-only manner
	// Overwrites only if block with same hash already stored
	StoreBlock(block *block.Block) error

	// StoreCandidateBlock stores a candidate block to be proposed in next
	// consensus round.
	StoreCandidateBlock(block *block.Block) error

	// DeleteCandidateBlocks deletes all candidate blocks. If maxHeight is not
	// 0, it deletes only blocks with a height lower than maxHeight or equal. It
	// returns number of deleted candidate blocks
	DeleteCandidateBlocks(maxHeight uint64) (uint32, error)

	FetchBlock(hash []byte) (*block.Block, error)

	FetchCurrentHeight() (uint64, error)

	FetchDecoys(numDecoys int) []ristretto.Point

	FetchOutputExists(destkey []byte) (bool, error)

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

// State represents a single db entry that provides chain metadata. This
// includes currently only chain tip hash but could be extended at later stage
type State struct {
	TipHash []byte
}
