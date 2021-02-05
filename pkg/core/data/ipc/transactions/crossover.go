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
	ValueComm     []byte `json:"value_comm"`
	Nonce         []byte `json:"nonce"`
	EncryptedData []byte `json:"encrypted_data"`
}

// NewCrossover returns a new empty Crossover struct.
func NewCrossover() *Crossover {
	return &Crossover{
		ValueComm:     make([]byte, 32),
		Nonce:         make([]byte, 32),
		EncryptedData: make([]byte, 96),
	}
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (c *Crossover) Copy() *Crossover {
	valueComm := make([]byte, len(c.ValueComm))
	nonce := make([]byte, len(c.Nonce))
	encData := make([]byte, len(c.EncryptedData))

	copy(valueComm, c.ValueComm)
	copy(nonce, c.Nonce)
	copy(encData, c.EncryptedData)

	return &Crossover{
		ValueComm:     valueComm,
		Nonce:         nonce,
		EncryptedData: encData,
	}
}

// MarshalCrossover writes the Crossover struct into a bytes.Buffer.
func MarshalCrossover(r *bytes.Buffer, f *Crossover) error {
	if err := encoding.Write256(r, f.ValueComm); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.Nonce); err != nil {
		return err
	}

	return encoding.WriteVarBytes(r, f.EncryptedData)
}

// UnmarshalCrossover reads a Crossover struct from a bytes.Buffer.
func UnmarshalCrossover(r *bytes.Buffer, f *Crossover) error {
	if err := encoding.Read256(r, f.ValueComm); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.Nonce); err != nil {
		return err
	}

	return encoding.ReadVarBytes(r, &f.EncryptedData)
}

// Equal returns whether or not two Crossovers are equal.
func (c *Crossover) Equal(other *Crossover) bool {
	if !bytes.Equal(c.ValueComm, other.ValueComm) {
		return false
	}

	if !bytes.Equal(c.Nonce, other.Nonce) {
		return false
	}

	return bytes.Equal(c.EncryptedData, other.EncryptedData)
}
