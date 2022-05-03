// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package mempool

import (
	"errors"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
)

var errNotFound = errors.New("not found")

type txHash [32]byte

// TxDesc encapsulates both tx raw and meta data.
type TxDesc struct {
	tx transactions.ContractCall

	// the point in time, tx was received in pending queue.
	received time.Time
	// the point in time, tx was moved into verified pool.
	verified time.Time
	size     uint

	// Kadcast transport-specific field.
	kadHeight byte
}

// Pool represents a transaction pool of the verified txs only.
type Pool interface {
	// Create instantiates the underlying data storage.
	Create(path string) error
	// Put sets the value for the given key. It overwrites any previous value
	// for that key.
	Put(t TxDesc) error
	// Get retrieves a transaction for a given txID, if it exists.
	Get(txID []byte) transactions.ContractCall

	// GetTxsByNullifier returns a set of hashes of all transactions that
	// contain a given nullifier.
	GetTxsByNullifier(nullifier []byte) ([][]byte, error)

	// Contains returns true if the given key is in the pool.
	Contains(key []byte) bool
	// Delete a key in the pool.
	Delete(key []byte) error
	// Clone the entire pool.
	Clone() []transactions.ContractCall

	// FilterByType returns all verified transactions for a specific type.
	FilterByType(transactions.TxType) []transactions.ContractCall

	// Size is total number of bytes of all txs marshaling size.
	Size() uint32
	// Len returns the number of tx entries.
	Len() int

	// Range iterates through all tx entries.
	Range(fn func(k txHash, t TxDesc) error) error

	// RangeSort iterates through all tx entries sorted by Fee
	// in a descending order.
	RangeSort(fn func(k txHash, t TxDesc) (bool, error)) error

	// Close closes backend.
	Close()
}
