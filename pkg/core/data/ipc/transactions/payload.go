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
	Data []byte `json:"payload"`
}

// NewTransactionPayload returns a new empty TransactionPayload struct.
func NewTransactionPayload() *TransactionPayload {
	return &TransactionPayload{
		Data: make([]byte, 0),
	}
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (t *TransactionPayload) Copy() *TransactionPayload {
	payload := make([]byte, len(t.Data))
	copy(payload, t.Data)

	return &TransactionPayload{
		Data: payload,
	}
}

// MarshalTransactionPayload writes the TransactionPayload struct into a bytes.Buffer.
func MarshalTransactionPayload(r *bytes.Buffer, f *TransactionPayload) error {
	return encoding.WriteVarBytes(r, f.Data)
}

// UnmarshalTransactionPayload reads a TransactionPayload struct from a bytes.Buffer.
func UnmarshalTransactionPayload(r *bytes.Buffer, f *TransactionPayload) error {
	return encoding.ReadVarBytes(r, &f.Data)
}

// Equal returns whether or not two TransactionPayloads are equal.
func (t *TransactionPayload) Equal(other *TransactionPayload) bool {
	return bytes.Equal(t.Data, other.Data)
}
