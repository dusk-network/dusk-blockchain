// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package encoding

import (
	"bytes"
	"encoding/binary"
	"errors"
)

const (
	// IDLen PeerInfo ID length.
	IDLen = 16

	// NonceLen PoW-Nonce length.
	NonceLen = 4

	// Message types handled by Maintainer.

	// PingMsg wire Ping message id.
	PingMsg = 0

	// PongMsg wire Pong message id.
	PongMsg = 1

	// FindNodesMsg wire FindNodes message id.
	FindNodesMsg = 2

	// NodesMsg wire Nodes message id.
	NodesMsg = 3

	// Message types handled by (TCP) Reader or RaptorCodeReader.

	// BroadcastMsg Message propagation type.
	BroadcastMsg = 10
)

var byteOrder = binary.LittleEndian

// BinaryMarshaler interface for marshal/unmarshal wire unit.
type BinaryMarshaler interface {
	// Marshal payload to binary.
	MarshalBinary(buf *bytes.Buffer) error

	// Unmarshal payload from binary.
	UnmarshalBinary(buf *bytes.Buffer) error
}

// MarshalBinary marshals message into binary buffer.
func MarshalBinary(header Header, payload BinaryMarshaler, buf *bytes.Buffer) error {
	if err := header.MarshalBinary(buf); err != nil {
		return err
	}

	if payload != nil {
		if err := payload.MarshalBinary(buf); err != nil {
			return err
		}
	} else {
		switch header.MsgType {
		case PingMsg, PongMsg, FindNodesMsg:
			return nil
		default:
			return errors.New("missing message payload")
		}
	}

	return nil
}
