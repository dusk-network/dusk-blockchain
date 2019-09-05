package transactions

// TxType defines a transaction type identifier.
type TxType uint8

const (
	// CoinbaseType is the identifier for a block coinbase
	CoinbaseType TxType = 0x00

	// BidType is the identifier for a blind bid
	BidType TxType = 0x01

	// StakeType is the identifier for a stake
	StakeType TxType = 0x02

	// StandardType is the identifier for a standard transaction
	StandardType TxType = 0x03

	// TimelockType is the identifier for a standard time-locked transaction
	TimelockType TxType = 0x04

	// ContractType is the identifier for a smart contract transaction
	ContractType TxType = 0x05
)

var _ Transaction = (*Coinbase)(nil)
var _ Transaction = (*Bid)(nil)
var _ Transaction = (*Stake)(nil)
var _ Transaction = (*Standard)(nil)
var _ Transaction = (*Timelock)(nil)
