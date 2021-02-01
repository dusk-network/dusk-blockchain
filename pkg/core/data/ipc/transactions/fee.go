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

// Fee is a Phoenix fee note.
type Fee struct {
	GasLimit uint64 `json:"gas_limit"`
	GasPrice uint64 `json:"gas_price"`
	R        []byte `json:"r"`
	PkR      []byte `json:"pk_r"`
}

// NewFee returns a new empty Fee struct.
func NewFee() *Fee {
	return &Fee{
		R:   make([]byte, 32),
		PkR: make([]byte, 32),
	}
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (f *Fee) Copy() *Fee {
	r := make([]byte, len(f.R))
	pkR := make([]byte, len(f.PkR))

	copy(r, f.R)
	copy(pkR, f.PkR)

	return &Fee{
		GasLimit: f.GasLimit,
		GasPrice: f.GasPrice,
		R:        r,
		PkR:      pkR,
	}
}

// MarshalFee writes the Fee struct into a bytes.Buffer.
func MarshalFee(r *bytes.Buffer, f *Fee) error {
	if err := encoding.WriteUint64LE(r, f.GasLimit); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, f.GasPrice); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.R); err != nil {
		return err
	}

	return encoding.Write256(r, f.PkR)
}

// UnmarshalFee reads a Fee struct from a bytes.Buffer.
func UnmarshalFee(r *bytes.Buffer, f *Fee) error {
	if err := encoding.ReadUint64LE(r, &f.GasLimit); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &f.GasPrice); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.R); err != nil {
		return err
	}

	return encoding.Read256(r, f.PkR)
}

// Equal returns whether or not two Fees are equal.
func (f *Fee) Equal(other *Fee) bool {
	if f.GasLimit != other.GasLimit {
		return false
	}

	if f.GasPrice != other.GasPrice {
		return false
	}

	if !bytes.Equal(f.R, other.R) {
		return false
	}

	return bytes.Equal(f.PkR, other.PkR)
}
