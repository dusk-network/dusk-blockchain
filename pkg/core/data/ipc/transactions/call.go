// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package transactions

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

// Call represents a contract call.
type Call struct {
	// ContractID are 32 bytes representing the address of a contract. It is a valid BlsScalar.
	ContractID []byte

	// FnName is the name the name of the function to call (as bytes)
	FnName []byte
	// The data to call the contract with.
	CallData []byte
}

// NewCall returns a new empty Call struct.
func NewCall() *Call {
	return &Call{
		ContractID: make([]byte, 32),
		FnName:     make([]byte, 0),
		CallData:   make([]byte, 0),
	}
}

// UnmarshalCall reads a Call struct from a bytes.Buffer.
func UnmarshalCall(r *bytes.Buffer, c *Call) error {
	if err := encoding.Read256(r, c.ContractID); err != nil {
		return err
	}

	var lenFnName uint64
	if err := encoding.ReadUint64LE(r, &lenFnName); err != nil {
		return err
	}

	c.FnName = make([]byte, lenFnName)
	if _, err := io.ReadFull(r, c.FnName); err != nil {
		return err
	}

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	c.CallData = data

	return nil
}
