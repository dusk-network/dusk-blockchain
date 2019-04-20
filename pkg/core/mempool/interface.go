package mempool

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"time"
)

// TxDesc encapsulates both tx raw and meta data
type TxDesc struct {
	tx transactions.Transaction

	// the point in time tx was received in pending queue
	received time.Time
	// the point in time tx was moved into verified pool
	verified time.Time
	// the point in time tx was accepted by this node
	// accepted time.Time
}

// Limited set of accessors to force design constraints
type Pool interface {

	// Put sets the value for the given key. It overwrites any previous value
	// for that key; a Pool is not a multi-map.
	Put(t TxDesc) error
	// Contains returns true if the given key are in the pool.
	Contains(key []byte) bool
	// Clone the entire pool
	Clone() []transactions.Transaction

	// Pool sizing
	Size() uint32
	// Len returns the number of entries in the DB.
	Len() int

	// To range the pool items by a closure
	Range(fn func(k key, t TxDesc) error) error
}
