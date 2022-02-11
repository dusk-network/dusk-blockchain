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

// Crossover is the crossover note used in a Phoenix transaction.
type Crossover struct {
	ValueCommitment []byte `json:"value_comm"`
	Nonce           []byte `json:"nonce"`
	EncryptedData   []byte `json:"encrypted_data"`
}

// NewCrossover returns a new empty Crossover struct.
func NewCrossover() *Crossover {
	return &Crossover{
		ValueCommitment: make([]byte, 32),
		Nonce:           make([]byte, 32),
		EncryptedData:   make([]byte, 96),
	}
}

// MarshalCrossover writes the Crossover struct into a bytes.Buffer.
func MarshalCrossover(r *bytes.Buffer, f *Crossover) error {
	if err := encoding.Write256(r, f.ValueCommitment); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.Nonce); err != nil {
		return err
	}

	if _, err := r.Write(f.EncryptedData); err != nil {
		return err
	}

	return encoding.WriteVarBytes(r, f.EncryptedData)
}

// UnmarshalCrossover reads a Crossover struct from a bytes.Buffer.
func UnmarshalCrossover(r *bytes.Buffer, f *Crossover) error {
	if err := encoding.Read256(r, f.ValueCommitment); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.Nonce); err != nil {
		return err
	}

	if _, err := r.Read(f.EncryptedData); err != nil {
		return err
	}

	return nil
}
