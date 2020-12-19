// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package transactions

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/common"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// Fee is a Phoenix fee note.
type Fee struct {
	GasLimit uint64                   `json:"gas_limit"`
	GasPrice uint64                   `json:"gas_price"`
	R        *common.JubJubCompressed `json:"r"`
	PkR      *common.JubJubCompressed `json:"pk_r"`
}

// NewFee returns a new empty Fee struct.
func NewFee() *Fee {
	return &Fee{
		R:   common.NewJubJubCompressed(),
		PkR: common.NewJubJubCompressed(),
	}
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (f *Fee) Copy() *Fee {
	return &Fee{
		GasLimit: f.GasLimit,
		GasPrice: f.GasPrice,
		R:        f.R.Copy(),
		PkR:      f.PkR.Copy(),
	}
}

// MFee copies the Fee structure into the Rusk equivalent.
func MFee(r *rusk.Fee, f *Fee) {
	r.GasLimit = f.GasLimit
	r.GasPrice = f.GasPrice
	r.R = new(rusk.JubJubCompressed)
	common.MJubJubCompressed(r.R, f.R)
	r.PkR = new(rusk.JubJubCompressed)
	common.MJubJubCompressed(r.PkR, f.PkR)
}

// UFee copies the Rusk Fee structure into the native equivalent.
func UFee(r *rusk.Fee, f *Fee) {
	f.GasLimit = r.GasLimit
	f.GasPrice = r.GasPrice
	f.R = new(common.JubJubCompressed)
	common.UJubJubCompressed(r.R, f.R)
	f.PkR = new(common.JubJubCompressed)
	common.UJubJubCompressed(r.PkR, f.PkR)
}

// MarshalFee writes the Fee struct into a bytes.Buffer.
func MarshalFee(r *bytes.Buffer, f *Fee) error {
	if err := encoding.WriteUint64LE(r, f.GasLimit); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, f.GasPrice); err != nil {
		return err
	}

	if err := common.MarshalJubJubCompressed(r, f.R); err != nil {
		return err
	}

	return common.MarshalJubJubCompressed(r, f.PkR)
}

// UnmarshalFee reads a Fee struct from a bytes.Buffer.
func UnmarshalFee(r *bytes.Buffer, f *Fee) error {
	if err := encoding.ReadUint64LE(r, &f.GasLimit); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &f.GasPrice); err != nil {
		return err
	}

	if err := common.UnmarshalJubJubCompressed(r, f.R); err != nil {
		return err
	}

	return common.UnmarshalJubJubCompressed(r, f.PkR)
}

// Equal returns whether or not two Fees are equal.
func (f *Fee) Equal(other *Fee) bool {
	if f.GasLimit != other.GasLimit {
		return false
	}

	if f.GasPrice != other.GasPrice {
		return false
	}

	if !f.R.Equal(other.R) {
		return false
	}

	return f.PkR.Equal(other.PkR)
}
