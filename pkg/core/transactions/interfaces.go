package transactions

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/merkletree"
)

// Encoder encodes a given struct into an io.writer
type Encoder interface {
	Encode(io.Writer) error
}

// Decoder decodes a given io.Reader into a struct
type Decoder interface {
	Decode(io.Reader) error
}

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
	Encoder
	Decoder
	TypeInfo
	merkletree.Payload
	Equals(Transaction) bool
	StandardTX() Standard
}
