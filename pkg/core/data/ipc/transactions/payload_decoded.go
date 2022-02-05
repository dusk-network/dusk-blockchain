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

const size_BLS_SCALAR = 32

const size_CONTRACTID = 32

const capacity_MESSAGE = 2

const size_CIPHER = capacity_MESSAGE + 1

const size_CIPHER_BYTES = size_CIPHER * size_BLS_SCALAR

const size_NOTE = 137 + size_CIPHER_BYTES

const size_STEALTH_ADDRESS = 64

const size_G1AFFINE = 48

const size_COMMITMENT = size_G1AFFINE

const size_PROOF_EVAL = 24 * size_BLS_SCALAR

const size_PROOF = 15*size_COMMITMENT + size_PROOF_EVAL

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

// UnmarshalTransactionPayloadDecoded reads a TransactionPayloadDecoded struct from a bytes.Buffer.
func UnmarshalTransactionPayloadDecoded(r *bytes.Buffer, f *TransactionPayloadDecoded) error {
	var lenInputs uint64
	if err := encoding.ReadUint64LE(r, &lenInputs); err != nil {
		return err
	}

	f.Nullifiers = make([][]byte, lenInputs)
	for i := range f.Nullifiers {
		f.Nullifiers[i] = make([]byte, 32)
		if err := encoding.Read256(r, f.Nullifiers[i]); err != nil {
			return err
		}
	}

	var lenNotes uint64
	if err := encoding.ReadUint64LE(r, &lenNotes); err != nil {
		return err
	}

	f.Notes = make([]*Note, lenNotes)
	for i := range f.Notes {
		f.Notes[i] = NewNote()
		// TODO: read new notes format
		// if err := UnmarshalNote(r, f.Notes[i]); err != nil {
		// 	return err
		// }
		noteBuffer := make([]byte, size_NOTE)
		if _, err := r.Read(noteBuffer); err != nil {
			return err
		}
	}

	if err := encoding.Read256(r, f.Anchor); err != nil {
		return err
	}

	if err := UnmarshalFee(r, f.Fee); err != nil {
		return err
	}

	feeStealthAddress := make([]byte, size_STEALTH_ADDRESS)
	if _, err := r.Read(feeStealthAddress); err != nil {
		return err
	}

	f.SpendingProof = make([]byte, size_PROOF)
	if _, err := r.Read(f.SpendingProof); err != nil {
		return err
	}

	var crossoverFlag uint64
	if err := encoding.ReadUint64LE(r, &crossoverFlag); err != nil {
		return err
	}

	if crossoverFlag > 0 {
		if err := UnmarshalCrossover(r, f.Crossover); err != nil {
			return err
		}
	}

	var calldataFlag uint64
	if err := encoding.ReadUint64LE(r, &calldataFlag); err != nil {
		return err
	}

	if calldataFlag > 0 {
		// TODO: read f.CallData (there is no length prefix, just read the buffer until the end)
		contractID := make([]byte, size_CONTRACTID)
		if _, err := r.Read(contractID); err != nil {
			return err
		}
	}

	return nil
}
