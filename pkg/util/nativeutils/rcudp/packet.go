// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package rcudp

import (
	"bytes"
)

// Packet is the UDP packet that consists of encoding symbol data unit and raptor-specific data.
type Packet struct {
	messageID        msgID
	NumSourceSymbols uint16
	PaddingSize      uint16
	transferLength   uint32

	// payload fields
	blockID uint32
	block   [BlockSize]byte
}

func newPacket(msgID []byte, numSourceSymbols uint16,
	paddingSize uint16, transferLength uint32,
	blockID uint32, block []byte) Packet {
	p := Packet{
		NumSourceSymbols: numSourceSymbols,
		PaddingSize:      paddingSize,
		transferLength:   transferLength,
		blockID:          blockID,
	}

	copy(p.messageID[:], msgID)
	copy(p.block[:], block)

	return p
}

func (p *Packet) marshalBinary(buf *bytes.Buffer) error {
	// source object id
	if _, err := buf.Write(p.messageID[:]); err != nil {
		return err
	}

	// NumSourceSymbols
	b := make([]byte, 2)

	byteOrder.PutUint16(b, p.NumSourceSymbols)

	if _, err := buf.Write(b); err != nil {
		return err
	}

	// PaddingSize
	b = make([]byte, 2)

	byteOrder.PutUint16(b, p.PaddingSize)

	if _, err := buf.Write(b); err != nil {
		return err
	}

	// transferLength
	b = make([]byte, 4)

	byteOrder.PutUint32(b, p.transferLength)

	if _, err := buf.Write(b); err != nil {
		return err
	}

	// blockID
	b = make([]byte, 4)

	byteOrder.PutUint32(b, p.blockID)

	if _, err := buf.Write(b); err != nil {
		return err
	}

	if _, err := buf.Write(p.block[:]); err != nil {
		return err
	}

	if buf.Len() > maxPacketLen {
		return ErrTooLargeUDP
	}

	return nil
}

func (p *Packet) unmarshalBinary(buf *bytes.Buffer) error {
	objectID := make([]byte, 8)
	if _, err := buf.Read(objectID); err != nil {
		return err
	}

	numSourceSymbols := make([]byte, 2)
	if _, err := buf.Read(numSourceSymbols); err != nil {
		return err
	}

	PaddingSize := make([]byte, 2)
	if _, err := buf.Read(PaddingSize); err != nil {
		return err
	}

	transferLength := make([]byte, 4)
	if _, err := buf.Read(transferLength); err != nil {
		return err
	}

	blockID := make([]byte, 4)
	if _, err := buf.Read(blockID); err != nil {
		return err
	}

	block := make([]byte, buf.Len())
	if _, err := buf.Read(block); err != nil {
		return err
	}

	copy(p.messageID[:], objectID)
	copy(p.block[:], block)

	p.NumSourceSymbols = byteOrder.Uint16(numSourceSymbols)
	p.PaddingSize = byteOrder.Uint16(PaddingSize)
	p.transferLength = byteOrder.Uint32(transferLength)
	p.blockID = byteOrder.Uint32(blockID)

	return nil
}
