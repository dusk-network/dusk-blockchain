// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package transactions

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

// TransactionPayloadDecoded carries the common data contained in all transaction types.
type TransactionPayloadDecoded struct {
	Anchor        []byte   `json:"anchor"`
	Nullifiers    [][]byte `json:"nullifier"`
	*Crossover    `json:"crossover"`
	Notes         []*Note `json:"notes"`
	*Fee          `json:"fee"`
	SpendingProof []byte `json:"spending_proof"`
	CallData      []byte `json:"call_data"`
}

// NewTransactionPayloadDecoded returns a new empty TransactionPayloadDecoded struct.
func NewTransactionPayloadDecoded() *TransactionPayloadDecoded {
	return &TransactionPayloadDecoded{
		Anchor:        make([]byte, 32),
		Nullifiers:    make([][]byte, 0),
		Crossover:     NewCrossover(),
		Notes:         make([]*Note, 0),
		Fee:           NewFee(),
		SpendingProof: make([]byte, 0),
		CallData:      make([]byte, 0),
	}
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (t *TransactionPayloadDecoded) Copy() *TransactionPayloadDecoded {
	anchor := make([]byte, len(t.Anchor))
	copy(anchor, t.Anchor)

	inputs := make([][]byte, len(t.Nullifiers))
	for i := range inputs {
		inputs[i] = make([]byte, len(t.Nullifiers[i]))
		copy(inputs[i], t.Nullifiers[i])
	}

	notes := make([]*Note, len(t.Notes))
	for i := range notes {
		notes[i] = t.Notes[i].Copy()
	}

	spendingProof := make([]byte, len(t.SpendingProof))
	copy(spendingProof, t.SpendingProof)

	callData := make([]byte, len(t.CallData))
	copy(callData, t.CallData)

	return &TransactionPayloadDecoded{
		Anchor:        anchor,
		Nullifiers:    inputs,
		Crossover:     t.Crossover.Copy(),
		Notes:         notes,
		Fee:           t.Fee.Copy(),
		SpendingProof: spendingProof,
		CallData:      callData,
	}
}

// MarshalTransactionPayloadDecoded writes the TransactionPayloadDecoded struct into a bytes.Buffer.
func MarshalTransactionPayloadDecoded(r *bytes.Buffer, f *TransactionPayloadDecoded) error {
	if err := encoding.Write256(r, f.Anchor); err != nil {
		return err
	}

	if err := encoding.WriteVarInt(r, uint64(len(f.Nullifiers))); err != nil {
		return err
	}

	for _, input := range f.Nullifiers {
		if err := encoding.Write256(r, input); err != nil {
			return err
		}
	}

	if err := MarshalCrossover(r, f.Crossover); err != nil {
		return err
	}

	if err := encoding.WriteVarInt(r, uint64(len(f.Notes))); err != nil {
		return err
	}

	for _, note := range f.Notes {
		if err := MarshalNote(r, note); err != nil {
			return err
		}
	}

	if err := MarshalFee(r, f.Fee); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, f.SpendingProof); err != nil {
		return err
	}

	return encoding.WriteVarBytes(r, f.CallData)
}

// UnmarshalTransactionPayloadDecoded reads a TransactionPayloadDecoded struct from a bytes.Buffer.
func UnmarshalTransactionPayloadDecoded(r *bytes.Buffer, f *TransactionPayloadDecoded) error {
	if err := encoding.Read256(r, f.Anchor); err != nil {
		return err
	}

	lenInputs, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	f.Nullifiers = make([][]byte, lenInputs)
	for i := range f.Nullifiers {
		f.Nullifiers[i] = make([]byte, 32)
		if err = encoding.Read256(r, f.Nullifiers[i]); err != nil {
			return err
		}
	}

	if err = UnmarshalCrossover(r, f.Crossover); err != nil {
		return err
	}

	lenNotes, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	f.Notes = make([]*Note, lenNotes)
	for i := range f.Notes {
		f.Notes[i] = NewNote()
		if err = UnmarshalNote(r, f.Notes[i]); err != nil {
			return err
		}
	}

	if err = UnmarshalFee(r, f.Fee); err != nil {
		return err
	}

	if err = encoding.ReadVarBytes(r, &f.SpendingProof); err != nil {
		return err
	}

	return encoding.ReadVarBytes(r, &f.CallData)
}

// Equal returns whether or not two Decoded TransactionPayloads are equal.
func (t *TransactionPayloadDecoded) Equal(other *TransactionPayloadDecoded) bool {
	if !bytes.Equal(t.Anchor, other.Anchor) {
		return false
	}

	if len(t.Nullifiers) != len(other.Nullifiers) {
		return false
	}

	for i := range t.Nullifiers {
		if !bytes.Equal(t.Nullifiers[i], other.Nullifiers[i]) {
			return false
		}
	}

	if !t.Crossover.Equal(other.Crossover) {
		return false
	}

	if len(t.Notes) != len(other.Notes) {
		return false
	}

	for i := range t.Notes {
		if !t.Notes[i].Equal(other.Notes[i]) {
			return false
		}
	}

	if !bytes.Equal(t.SpendingProof, other.SpendingProof) {
		return false
	}

	return bytes.Equal(t.CallData, other.CallData)
}
