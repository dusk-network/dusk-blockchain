package transactions

import "github.com/dusk-network/dusk-crypto/merkletree"

// TypeInfo returns the underlying type of the transaction
// This allows the caller to type cast to a
// TypeInfo and switch on the Types in order to Decode into a TX
type TypeInfo interface {
	Type() TxType
}

// Transaction represents all transaction structures
// All transactions will embed the standard transaction.
// Returning Standard() allows the caller to
// fetch the inputs/outputs/fees without type casting the
// Transaction interface to a specific type
type Transaction interface {
	TypeInfo
	merkletree.Payload
	Equals(Transaction) bool
	StandardTx() *Standard
}
