package transactions

// TxType defines a transaction type identifier.
type TxType uint8

const (
	// StandardType is the identifier for standard transactions.
	StandardType TxType = 0x00

	// BidType is the identifier for a blind bid
	BidType TxType = 0x01
)
