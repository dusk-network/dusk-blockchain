// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package transactions

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
	"github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-crypto/merkletree"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// TxType is the type identifier for a transaction.
type TxType uint32

const (
	// Tx indicates the phoenix transaction type.
	Tx TxType = iota
	// Distribute indicates the coinbase and reward distribution contract call.
	Distribute
	// WithdrawFees indicates the Provisioners' withdraw contract call.
	WithdrawFees
	// Bid transaction propagated by the Block Generator.
	Bid
	// Stake transaction propagated by the Provisioners.
	Stake
	// Slash transaction propagated by the consensus to punish the Committee
	// members when they turn byzantine.
	Slash
	// WithdrawStake transaction propagated by the Provisioners to withdraw
	// their stake.
	WithdrawStake
	// WithdrawBid transaction propagated by the Block Generator to withdraw
	// their bids.
	WithdrawBid
)

// Transaction is a Phoenix transaction.
type Transaction struct {
	Version uint32 `json:"version"`
	TxType  `json:"type"`
	Payload *TransactionPayload `json:"payload"`
}

// NewTransaction returns a new empty Transaction struct.
func NewTransaction() *Transaction {
	t := new(Transaction)
	t.Payload = NewTransactionPayload()
	return t
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (t Transaction) Copy() payload.Safe {
	return &Transaction{
		Version: t.Version,
		TxType:  t.TxType,
		Payload: t.Payload.Copy(),
	}
}

// MTransaction copies the Transaction structure into the Rusk equivalent.
func MTransaction(r *rusk.Transaction, f *Transaction) error {
	r.Version = f.Version
	r.Type = uint32(f.TxType)

	buf := new(bytes.Buffer)
	if err := MarshalTransactionPayload(buf, f.Payload); err != nil {
		return err
	}

	r.Payload = buf.Bytes()
	return nil
}

// UTransaction copies the Rusk Transaction structure into the native equivalent.
func UTransaction(r *rusk.Transaction, f *Transaction) error {
	f.Version = r.Version
	f.TxType = TxType(r.Type)
	f.Payload = NewTransactionPayload()

	buf := bytes.NewBuffer(r.Payload)
	return UnmarshalTransactionPayload(buf, f.Payload)
}

// MarshalTransaction writes the Transaction struct into a bytes.Buffer.
func MarshalTransaction(r *bytes.Buffer, f *Transaction) error {
	if err := encoding.WriteUint32LE(r, f.Version); err != nil {
		return err
	}

	if err := encoding.WriteUint32LE(r, uint32(f.TxType)); err != nil {
		return err
	}

	return MarshalTransactionPayload(r, f.Payload)
}

// UnmarshalTransaction reads a Transaction struct from a bytes.Buffer.
func UnmarshalTransaction(r *bytes.Buffer, f *Transaction) error {
	if err := encoding.ReadUint32LE(r, &f.Version); err != nil {
		return err
	}

	var t uint32
	if err := encoding.ReadUint32LE(r, &t); err != nil {
		return err
	}

	f.TxType = TxType(t)
	return UnmarshalTransactionPayload(r, f.Payload)
}

// ContractCall is the transaction that embodies the execution parameter for a
// smart contract method invocation.
type ContractCall interface {
	payload.Safe
	merkletree.Payload

	// StandardTx returns the payload.
	StandardTx() *TransactionPayload

	// Type indicates the transaction.
	Type() TxType

	// Obfuscated returns true if the TransactionOutputs consists of Obfuscated
	// Notes, false if they are transparent.
	Obfuscated() bool

	// Values returns a tuple where the first element is the sum of all transparent
	// outputs' note values and the second is the fee.
	Values() (amount uint64, fee uint64)
}

// Marshal a Contractcall to a bytes.Buffer.
func Marshal(r *bytes.Buffer, f ContractCall) error {
	switch f := f.(type) {
	case *Transaction:
		return MarshalTransaction(r, f)
	default:
		return errors.New("unrecognized type of ContractCall")
	}
}

// Unmarshal a ContractCall from a bytes.Buffer.
func Unmarshal(r *bytes.Buffer, f ContractCall) error {
	switch f := f.(type) {
	case *Transaction:
		return UnmarshalTransaction(r, f)
	default:
		return errors.New("unrecognized type of ContractCall")
	}
}

// StandardTx returns the transaction payload.
func (t Transaction) StandardTx() *TransactionPayload {
	return t.Payload
}

// CalculateHash returns the SHA3-256 hash digest of the transaction.
// TODO: this needs to correspond with how rusk hashes transactions.
func (t Transaction) CalculateHash() ([]byte, error) {
	b := new(bytes.Buffer)
	if err := Marshal(b, &t); err != nil {
		return nil, err
	}

	return hash.Sha3256(b.Bytes())
}

// Type returns the transaction type.
func (t Transaction) Type() TxType {
	return t.TxType
}

// Obfuscated returns whether or not the outputs of this transaction
// are obfuscated.
// TODO: implement.
func (t Transaction) Obfuscated() bool {
	return true
}

// Values returns the amount and fee spent in this transaction.
// TODO: add amount calculation.
func (t Transaction) Values() (amount uint64, fee uint64) {
	return 0, t.Payload.Fee.GasLimit * t.Payload.Fee.GasPrice
}

// Equal checks equality between two transactions.
// TODO: implement.
func Equal(t, other ContractCall) bool {
	if t.Type() != other.Type() {
		return false
	}

	return t.StandardTx().Equal(other.StandardTx())
}

// BidTransaction is used to perform a blind bid.
type BidTransaction struct {
	BidTreeStorageIndex uint64
	Tx                  *Transaction `json:"tx"`
}

// NewBidTransaction creates a new empty BidTransaction struct.
func NewBidTransaction() *BidTransaction {
	return &BidTransaction{
		Tx: NewTransaction(),
	}
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (b *BidTransaction) Copy() *BidTransaction {
	return &BidTransaction{
		BidTreeStorageIndex: b.BidTreeStorageIndex,
		Tx:                  b.Tx.Copy().(*Transaction),
	}
}

// MBidTransaction copies the BidTransaction structure into the Rusk equivalent.
func MBidTransaction(r *rusk.BidTransaction, f *BidTransaction) error {
	r.BidTreeStorageIndex = f.BidTreeStorageIndex

	return MTransaction(r.Tx, f.Tx)
}

// UBidTransaction copies the Rusk BidTransaction structure into the native equivalent.
func UBidTransaction(r *rusk.BidTransaction, f *BidTransaction) error {
	f.BidTreeStorageIndex = r.BidTreeStorageIndex

	return UTransaction(r.Tx, f.Tx)
}

// MarshalBidTransaction writes the BidTransaction struct into a bytes.Buffer.
func MarshalBidTransaction(r *bytes.Buffer, f *BidTransaction) error {
	if err := encoding.WriteUint64LE(r, f.BidTreeStorageIndex); err != nil {
		return err
	}

	return MarshalTransaction(r, f.Tx)
}

// UnmarshalBidTransaction reads a BidTransaction struct from a bytes.Buffer.
func UnmarshalBidTransaction(r *bytes.Buffer, f *BidTransaction) error {
	if err := encoding.ReadUint64LE(r, &f.BidTreeStorageIndex); err != nil {
		return err
	}

	return UnmarshalTransaction(r, f.Tx)
}
