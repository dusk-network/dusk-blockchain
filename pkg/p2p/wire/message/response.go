// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package message

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
)

// Response to a voucher seeder challenge.
type Response struct {
	HashedChallenge []byte
	Port            string
}

// Copy a Response.
// Implements the payload.Safe interface.
func (r Response) Copy() payload.Safe {
	b := make([]byte, len(r.HashedChallenge))
	copy(b, r.HashedChallenge)
	return Response{HashedChallenge: b, Port: r.Port}
}

// Encode a Response object into a buffer.
func (r *Response) Encode(w *bytes.Buffer) error {
	if err := encoding.WriteVarBytes(w, r.HashedChallenge); err != nil {
		return err
	}

	return encoding.WriteString(w, r.Port)
}

// UnmarshalResponseMessage into a SerializableMessage.
func UnmarshalResponseMessage(r *bytes.Buffer, m SerializableMessage) error {
	resp := &Response{}
	if err := resp.Decode(r); err != nil {
		return err
	}

	m.SetPayload(*resp)
	return nil
}

// Decode a Response message from a buffer.
func (r *Response) Decode(b *bytes.Buffer) error {
	hc := make([]byte, 0)
	if err := encoding.ReadVarBytes(b, &hc); err != nil {
		return err
	}

	port, err := encoding.ReadString(b)
	if err != nil {
		return err
	}

	r.HashedChallenge = hc
	r.Port = port
	return nil
}
