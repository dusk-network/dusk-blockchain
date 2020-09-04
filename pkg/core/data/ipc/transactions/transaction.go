package transactions

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-crypto/merkletree"
)

type TxType uint32

const (
	// Tx indicates the phoenix transaction type
	Tx TxType = iota
	// Distribute indicates the coinbase and reward distribution contract call
	Distribute
	// WithdrawFees indicates the Provisioners' withdraw contract call
	WithdrawFees
	// Bid transaction propagated by the Block Generator
	Bid
	// Stake transaction propagated by the Provisioners
	Stake
	// Slash transaction propagated by the consensus to punish the Committee
	// members when they turn byzantine
	Slash
	// WithdrawStake transaction propagated by the Provisioners to withdraw
	// their stake
	WithdrawStake
	// WithdrawBid transaction propagated by the Block Generator to withdraw
	// their bids
	WithdrawBid
)

// Transaction is a Phoenix transaction.
type Transaction struct {
	Version   uint32 `json:"version"`
	TxType    `json:"type"`
	TxPayload *TransactionPayload `json:"tx_payload"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (t Transaction) Copy() payload.Safe {
	return &Transaction{
		Version:   t.Version,
		TxType:    t.TxType,
		TxPayload: t.TxPayload.Copy(),
	}
}

// MTransaction copies the Transaction structure into the Rusk equivalent.
func MTransaction(r *rusk.Transaction, f *Transaction) {
	r.Version = f.Version
	r.Type = uint32(f.TxType)
	MTransactionPayload(r.TxPayload, f.TxPayload)
}

// UTransaction copies the Rusk Transaction structure into the native equivalent.
func UTransaction(r *rusk.Transaction, f *Transaction) {
	f.Version = r.Version
	f.TxType = TxType(r.Type)
	UTransactionPayload(r.TxPayload, f.TxPayload)
}

// MarshalTransaction writes the Transaction struct into a bytes.Buffer.
func MarshalTransaction(r *bytes.Buffer, f *Transaction) error {
	if err := encoding.WriteUint32LE(r, f.Version); err != nil {
		return err
	}

	if err := encoding.WriteUint32LE(r, uint32(f.TxType)); err != nil {
		return err
	}

	return MarshalTransactionPayload(r, f.TxPayload)
}

// UnmarshalTransaction reads a Transaction struct from a bytes.Buffer.
func UnmarshalTransaction(r *bytes.Buffer, f *Transaction) error {
	f = new(Transaction)

	if err := encoding.ReadUint32LE(r, &f.Version); err != nil {
		return err
	}

	var t uint32
	if err := encoding.ReadUint32LE(r, &t); err != nil {
		return err
	}
	f.TxType = TxType(t)

	return UnmarshalTransactionPayload(r, f.TxPayload)
}

// ContractCall is the transaction that embodies the execution parameter for a
// smart contract method invocation
type ContractCall interface {
	payload.Safe
	merkletree.Payload

	// StandardTx returns the payload
	StandardTx() *TransactionPayload

	// Type indicates the transaction
	Type() TxType

	// Obfuscated returns true if the TransactionOutputs consists of Obfuscated
	// Notes, false if they are transparent
	Obfuscated() bool

	// Values returns a tuple where the first element is the sum of all transparent
	// outputs' note values and the second is the fee
	Values() (amount uint64, fee uint64)
}

func Marshal(r *bytes.Buffer, f ContractCall) error {
	switch f.(type) {
	case *Transaction:
		return MarshalTransaction(r, f.(*Transaction))
	default:
		return errors.New("")
	}
}

func Unmarshal(r *bytes.Buffer, f ContractCall) error {
	switch f.(type) {
	case *Transaction:
		return UnmarshalTransaction(r, f.(*Transaction))
	default:
		return errors.New("")
	}
}

func (t Transaction) StandardTx() *TransactionPayload {
	return t.TxPayload
}

func (t Transaction) CalculateHash() ([]byte, error) {
	b := new(bytes.Buffer)
	if err := Marshal(b, &t); err != nil {
		return nil, err
	}

	return hash.Sha3256(b.Bytes())
}

func (t Transaction) Type() TxType {
	return t.TxType
}

// TODO: implement
func (t Transaction) Obfuscated() bool {
	return true
}

// TODO: implement
func (t Transaction) Values() (amount uint64, fee uint64) {
	return
}

// TODO: implement
func Equal(t, other ContractCall) bool {
	return false
}
