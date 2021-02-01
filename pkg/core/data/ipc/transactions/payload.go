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

// TransactionPayload carries the common data contained in all transaction types.
type TransactionPayload struct {
	Anchor        []byte   `json:"anchor"`
	Nullifiers    [][]byte `json:"nullifier"`
	*Crossover    `json:"crossover"`
	Notes         []*Note `json:"notes"`
	*Fee          `json:"fee"`
	SpendingProof []byte `json:"spending_proof"`
	CallData      []byte `json:"call_data"`
}

// NewTransactionPayload returns a new empty TransactionPayload struct.
func NewTransactionPayload() *TransactionPayload {
	return &TransactionPayload{
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
func (t *TransactionPayload) Copy() *TransactionPayload {
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

	return &TransactionPayload{
		Anchor:        anchor,
		Nullifiers:    inputs,
		Crossover:     t.Crossover.Copy(),
		Notes:         notes,
		Fee:           t.Fee.Copy(),
		SpendingProof: spendingProof,
		CallData:      callData,
	}
}

// MarshalTransactionPayload writes the TransactionPayload struct into a bytes.Buffer.
func MarshalTransactionPayload(r *bytes.Buffer, f *TransactionPayload) error {
	if err := encoding.Write256(r, f.Anchor); err != nil {
		return err
	}

	if err := encoding.WriteVarInt(r, uint64(len(f.Nullifiers))); err != nil {
		return err
	}

	for _, input := range f.Nullifiers {
		// XXX: We are using variable size bytes here, because we abuse the
		// input when mocking RUSK, to fill it with lots of data.
		// This should be fixed when we integrate the actual RUSK server.
		if err := encoding.WriteVarBytes(r, input); err != nil {
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

// UnmarshalTransactionPayload reads a TransactionPayload struct from a bytes.Buffer.
func UnmarshalTransactionPayload(r *bytes.Buffer, f *TransactionPayload) error {
	if err := encoding.Read256(r, f.Anchor); err != nil {
		return err
	}

	lenInputs, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	f.Nullifiers = make([][]byte, lenInputs)
	for i := range f.Nullifiers {
		// XXX: Change back to normal once RUSK is integrated.
		f.Nullifiers[i] = make([]byte, 0)
		if err = encoding.ReadVarBytes(r, &f.Nullifiers[i]); err != nil {
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

// Equal returns whether or not two TransactionPayloads are equal.
func (t *TransactionPayload) Equal(other *TransactionPayload) bool {
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
