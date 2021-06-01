// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package encoding

import (
	"bytes"
	"errors"
)

const (
	// HeaderFixedLength number of header bytes.
	HeaderFixedLength = 25
)

// Header represents the header part of kadcast wire messages. Both TCP and
// UDP are sharing same header structure.
type Header struct {
	MsgType byte

	// Remote peer details.
	RemotePeerID    [IDLen]byte
	RemotePeerNonce uint32
	RemotePeerPort  uint16

	// Reserved bytes for future use.
	Reserved [2]byte
}

// MarshalBinary marshal wire header into bytes buffer, if valid.
func (h *Header) MarshalBinary(buf *bytes.Buffer) error {
	if _, err := MsgTypeToString(h.MsgType); err != nil {
		return errors.New("unknown message type")
	}

	senderNonce := make([]byte, NonceLen)
	byteOrder.PutUint32(senderNonce, h.RemotePeerNonce)

	// Verify Nonce-PoW
	if err := verifyIDNonce(h.RemotePeerID, senderNonce); err != nil {
		return err
	}

	port := make([]byte, 2)
	byteOrder.PutUint16(port, h.RemotePeerPort)

	if err := buf.WriteByte(h.MsgType); err != nil {
		return err
	}

	if _, err := buf.Write(h.RemotePeerID[:]); err != nil {
		return err
	}

	if _, err := buf.Write(senderNonce); err != nil {
		return err
	}

	if _, err := buf.Write(port); err != nil {
		return err
	}

	if _, err := buf.Write(h.Reserved[:]); err != nil {
		return err
	}

	return nil
}

// UnmarshalBinary umarshal wire header from bytes buffer, if valid
// It ensures ID-nonce pair is correct.
func (h *Header) UnmarshalBinary(buf *bytes.Buffer) error {
	msgType, err := buf.ReadByte()
	if err != nil {
		return err
	}

	if _, err := MsgTypeToString(msgType); err != nil {
		return errors.New("unknown message type")
	}

	var senderID [IDLen]byte
	if _, err := buf.Read(senderID[:]); err != nil {
		return err
	}

	nonce := make([]byte, NonceLen)
	if _, err := buf.Read(nonce); err != nil {
		return err
	}

	port := make([]byte, 2)
	if _, err := buf.Read(port); err != nil {
		return err
	}

	reserved := make([]byte, 2)
	if _, err := buf.Read(reserved); err != nil {
		return err
	}

	if err := verifyIDNonce(senderID, nonce); err != nil {
		return err
	}

	h.MsgType = msgType
	h.RemotePeerID = senderID
	h.RemotePeerNonce = byteOrder.Uint32(nonce)
	h.RemotePeerPort = byteOrder.Uint16(port)
	copy(h.Reserved[:], reserved)

	return nil
}
