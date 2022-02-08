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
	GasLimit    uint64 `json:"gas_limit"`
	GasPrice    uint64 `json:"gas_price"`
	StealthAddr []byte `json:"stealth_addr"`
}

// NewFee returns a new empty Fee struct.
func NewFee() *Fee {
	return &Fee{
		GasLimit:    0,
		GasPrice:    0,
		StealthAddr: make([]byte, 64),
	}
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (f *Fee) Copy() *Fee {
	return &Fee{
		GasLimit: f.GasLimit,
		GasPrice: f.GasPrice,
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

	return nil
}

// UnmarshalFee reads a Fee struct from a bytes.Buffer.
func UnmarshalFee(r *bytes.Buffer, f *Fee) error {
	if err := encoding.ReadUint64LE(r, &f.GasLimit); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &f.GasPrice); err != nil {
		return err
	}

	if err := encoding.Read512(r, f.StealthAddr); err != nil {
		return err
	}

	return nil
}

// Equal returns if the two Fees are equal.
func (f *Fee) Equal(other *Fee) bool {
	if f.GasLimit != other.GasLimit {
		return false
	}

	if f.GasPrice != other.GasPrice {
		return false
	}

	return bytes.Equal(f.StealthAddr, other.StealthAddr)
}
