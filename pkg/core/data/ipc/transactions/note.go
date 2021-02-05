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

// Note represents a Phoenix note.
type Note struct {
	Randomness    []byte `json:"randomness"`
	PkR           []byte `json:"pk_r"`
	Commitment    []byte `json:"commitment"`
	Nonce         []byte `json:"nonce"`
	EncryptedData []byte `json:"encrypted_data"`
}

// NewNote returns a new empty Note struct.
func NewNote() *Note {
	return &Note{
		Randomness:    make([]byte, 32),
		PkR:           make([]byte, 32),
		Commitment:    make([]byte, 32),
		Nonce:         make([]byte, 32),
		EncryptedData: make([]byte, 96),
	}
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (n *Note) Copy() *Note {
	randomness := make([]byte, len(n.Randomness))
	pkR := make([]byte, len(n.PkR))
	commitment := make([]byte, len(n.Commitment))
	nonce := make([]byte, len(n.Nonce))
	encData := make([]byte, len(n.EncryptedData))

	copy(randomness, n.Randomness)
	copy(pkR, n.PkR)
	copy(commitment, n.Commitment)
	copy(nonce, n.Nonce)
	copy(encData, n.EncryptedData)

	return &Note{
		Randomness:    randomness,
		PkR:           pkR,
		Commitment:    commitment,
		Nonce:         nonce,
		EncryptedData: encData,
	}
}

// MarshalNote writes the Note struct into a bytes.Buffer.
func MarshalNote(r *bytes.Buffer, f *Note) error {
	if err := encoding.Write256(r, f.Randomness); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.PkR); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.Commitment); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.Nonce); err != nil {
		return err
	}

	return encoding.WriteVarBytes(r, f.EncryptedData)
}

// UnmarshalNote reads a Note struct from a bytes.Buffer.
func UnmarshalNote(r *bytes.Buffer, f *Note) error {
	if err := encoding.Read256(r, f.Randomness); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.PkR); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.Commitment); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.Nonce); err != nil {
		return err
	}

	return encoding.ReadVarBytes(r, &f.EncryptedData)
}

// Equal returns whether or not two Notes are equal.
func (n *Note) Equal(other *Note) bool {
	if !bytes.Equal(n.Randomness, other.Randomness) {
		return false
	}

	if !bytes.Equal(n.PkR, other.PkR) {
		return false
	}

	if !bytes.Equal(n.Commitment, other.Commitment) {
		return false
	}

	if !bytes.Equal(n.Nonce, other.Nonce) {
		return false
	}

	return bytes.Equal(n.EncryptedData, other.EncryptedData)
}
