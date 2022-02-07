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

// Note types.
const (
	NoteTypeTransparent uint8 = 0
	NoteTypeObfuscated  uint8 = 1
)

// Note represents a Phoenix note.
type Note struct {
	Type            uint8  `json:"note_type"`
	ValueCommitment []byte `json:"value_commitment"`
	Nonce           []byte `json:"nonce"`
	StealthAddress  []byte `json:"stealth_address"`
	Pos             uint64 `json:"pos"`
	EncryptedData   []byte `json:"encrypted_data"`
}

// NewNote returns a new empty Note struct.
func NewNote() *Note {
	return &Note{
		Type:            NoteTypeTransparent,
		ValueCommitment: make([]byte, 32),
		Nonce:           make([]byte, 32),
		StealthAddress:  make([]byte, 64),
		Pos:             0,
		EncryptedData:   make([]byte, 96),
	}
}

// MarshalNote writes the Note struct into a bytes.Buffer.
func MarshalNote(r *bytes.Buffer, f *Note) error {
	if err := encoding.WriteUint8(r, f.Type); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.ValueCommitment); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.Nonce); err != nil {
		return err
	}

	if err := encoding.Write512(r, f.StealthAddress); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, f.Pos); err != nil {
		return err
	}

	if _, err := r.Write(f.EncryptedData); err != nil {
		return err
	}

	return nil
}

// UnmarshalNote reads a Note struct from a bytes.Buffer.
func UnmarshalNote(r *bytes.Buffer, f *Note) (err error) {
	if err := encoding.ReadUint8(r, &f.Type); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.ValueCommitment); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.Nonce); err != nil {
		return err
	}

	if err := encoding.Read512(r, f.StealthAddress); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &f.Pos); err != nil {
		return err
	}

	if _, err := r.Read(f.EncryptedData); err != nil {
		return err
	}

	return nil
}
