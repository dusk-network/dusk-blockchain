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
	"github.com/dusk-network/dusk-crypto/merkletree"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// TxType is the type identifier for a transaction.
type TxType uint32

const (
	// Tx indicates the phoenix transaction type.
	Tx TxType = iota
	// Distribute indicates the coinbase and reward distribution contract call.
	Distribute = 0
	// Transfer transaction id.
	Transfer = 1
	// Stake transaction id.
	Stake
)

// Transaction is a Phoenix transaction.
type Transaction struct {
	Version uint32 `json:"version"`
	TxType  `json:"type"`
	Payload *TransactionPayload `json:"payload"`

	// Extended
	Hash     [32]byte
	FeeValue Fee
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
		Version:  t.Version,
		TxType:   t.TxType,
		Payload:  t.Payload.Copy(),
		Hash:     t.Hash,
		FeeValue: t.FeeValue,
	}
}

// Fee returns GasLimit.
func (t Transaction) Fee() uint64 {
	return t.FeeValue.GasLimit
}

// CalculateHash returns hash of transaction, if set.
func (t Transaction) CalculateHash() ([]byte, error) {
	return t.Hash[:], nil
}

// MTransaction copies the Transaction structure into the Rusk equivalent.
func MTransaction(r *rusk.Transaction, f *Transaction) error {
	r.Version = f.Version
	r.Type = uint32(f.TxType)

	r.Payload = f.Payload.Data
	return nil
}

// UTransaction copies the Rusk Transaction structure into the native equivalent.
func UTransaction(r *rusk.Transaction, f *Transaction) error {
	f.Version = r.Version
	f.TxType = TxType(r.Type)
	f.Payload = NewTransactionPayload()
	f.Payload.Data = r.Payload

	return nil
}

// MarshalTransaction writes the Transaction struct into a bytes.Buffer.
func MarshalTransaction(r *bytes.Buffer, f *Transaction) error {
	if err := encoding.WriteUint32LE(r, f.Version); err != nil {
		return err
	}

	if err := encoding.WriteUint32LE(r, uint32(f.TxType)); err != nil {
		return err
	}

	if err := MarshalTransactionPayload(r, f.Payload); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.Hash[:]); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, f.FeeValue.GasLimit); err != nil {
		return err
	}

	return encoding.WriteUint64LE(r, f.FeeValue.GasPrice)
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
	if err := UnmarshalTransactionPayload(r, f.Payload); err != nil {
		return err
	}

	b := make([]byte, 32)
	if err := encoding.Read256(r, b); err != nil {
		return err
	}

	copy(f.Hash[:], b)

	var gaslimit uint64
	if err := encoding.ReadUint64LE(r, &gaslimit); err != nil {
		return err
	}

	f.FeeValue.GasLimit = gaslimit

	var gasPrice uint64
	if err := encoding.ReadUint64LE(r, &gasPrice); err != nil {
		return err
	}

	f.FeeValue.GasPrice = gasPrice
	return nil
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

	Fee() uint64
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

// Type returns the transaction type.
func (t Transaction) Type() TxType {
	return t.TxType
}

// Equal checks equality between two transactions.
func Equal(t, other ContractCall) bool {
	if t.Type() != other.Type() {
		return false
	}

	return t.StandardTx().Equal(other.StandardTx())
}

// Extend recreate transaction with optional fields initialized.
func Extend(t ContractCall, f Fee, hash []byte) (ContractCall, error) {
	switch t := t.(type) {
	case *Transaction:
		n := &Transaction{
			Version: t.Version,
			TxType:  t.TxType,
			Payload: t.Payload.Copy(),
		}

		n.FeeValue = f

		if len(hash) != len(n.Hash) {
			return nil, errors.New("invalid length")
		}

		copy(n.Hash[:], hash)
		return n, nil
	default:
		return nil, errors.New("unrecognized type of ContractCall")
	}
}

// Decode returns a TransactionPayloadDecoded.
func (t *Transaction) Decode() (*TransactionPayloadDecoded, error) {
	decoded := NewTransactionPayloadDecoded()

	buffer := bytes.NewBuffer(t.Payload.Data)
	if err := UnmarshalTransactionPayloadDecoded(buffer, decoded, t.TxType); err != nil {
		return nil, err
	}
	return decoded, nil
}
