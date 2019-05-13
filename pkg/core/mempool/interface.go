package mempool

import (
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

// TxDesc encapsulates both tx raw and meta data
type TxDesc struct {
	tx transactions.Transaction

	// the point in time, tx was received in pending queue
	received time.Time
	// the point in time, tx was moved into verified pool
	verified time.Time
	// the point in time, tx was accepted by this node
	// accepted time.Time
}

// Pool represents a transaction pool of the verified txs only.
type Pool interface {

	// Put sets the value for the given key. It overwrites any previous value
	// for that key;
	Put(t TxDesc) error
	// Contains returns true if the given key is in the pool.
	Contains(key []byte) bool
	// ContainsKeyImage returns true if txpool includes a input that contains
	// this keyImage
	ContainsKeyImage(keyImage []byte) bool
	// Clone the entire pool
	Clone() []transactions.Transaction

	// Pool sizing in KBs
	Size() float64
	// Len returns the number of tx entries
	Len() int

	// Range iterates through all tx entries
	Range(fn func(k key, t TxDesc) error) error
}
