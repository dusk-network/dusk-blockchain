// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package rcudp

import (
	"errors"
)

const (
	messageIDFieldSize        = 8
	numSourceSymbolsFieldSize = 2
	paddingSizeFieldSize      = 2
	transferLengthFieldSize   = 4
	blockIDFieldSize          = 4
	bcastHeightSize           = 1

	// BcastHeightPos
	BcastHeightPos = 20

	packetMinSize = messageIDFieldSize +
		numSourceSymbolsFieldSize +
		paddingSizeFieldSize +
		transferLengthFieldSize +
		blockIDFieldSize +
		bcastHeightSize
)

// Packet is the UDP packet that consists of encoding symbol data unit and raptor-specific data.
type Packet struct {
	messageID        msgID
	NumSourceSymbols uint16
	PaddingSize      uint16
	transferLength   uint32

	// payload fields
	blockID uint32
	block   []byte

	bcastHeight byte
}

func newPacket(msgID []byte, numSourceSymbols uint16,
	paddingSize uint16, transferLength uint32,
	blockID uint32, block []byte, bcastHeight byte) Packet {
	p := Packet{
		NumSourceSymbols: numSourceSymbols,
		PaddingSize:      paddingSize,
		transferLength:   transferLength,
		blockID:          blockID,
		bcastHeight:      bcastHeight,
		block:            block,
	}

	copy(p.messageID[:], msgID)
	return p
}

// marshal serializes a packet struct to a byte blob.
// Any mem alloc here would impact perf so bytes.Buffer is not used.
func (p *Packet) marshal() ([]byte, error) {
	blob := make([]byte, 0, packetMinSize+len(p.block))

	// source object id
	blob = append(blob, p.messageID[:]...)

	// NumSourceSymbols
	b := make([]byte, numSourceSymbolsFieldSize)
	byteOrder.PutUint16(b, p.NumSourceSymbols)
	blob = append(blob, b...)
	b[0] = 0
	b[1] = 0

	// PaddingSize
	byteOrder.PutUint16(b, p.PaddingSize)
	blob = append(blob, b...)

	// transferLength
	b = make([]byte, transferLengthFieldSize)
	byteOrder.PutUint32(b, p.transferLength)
	blob = append(blob, b...)

	// blockID
	b[0] = 0
	b[1] = 0
	b[2] = 0
	b[3] = 0
	byteOrder.PutUint32(b, p.blockID)
	blob = append(blob, b...)

	// BCAST_HEIGHT field
	blob = append(blob, p.bcastHeight)

	// block data
	blob = append(blob, p.block[:]...)

	if len(blob) > maxPacketLen {
		return nil, ErrTooLargeUDP
	}

	return blob, nil
}

// unmarshal constructs a packet struct from a byte blob.
// Any mem alloc here would impact perf so bytes.Buffer is not used.
func (p *Packet) unmarshal(buf []byte) error {
	if len(buf) < packetMinSize {
		return errors.New("invalid packet size")
	}

	if len(buf) > maxPacketLen {
		return ErrTooLargeUDP
	}

	offset := 0
	copy(p.messageID[:], buf[offset:len(p.messageID)])
	offset += len(p.messageID)

	p.NumSourceSymbols = byteOrder.Uint16(buf[offset : offset+numSourceSymbolsFieldSize])
	offset += numSourceSymbolsFieldSize

	p.PaddingSize = byteOrder.Uint16(buf[offset : offset+paddingSizeFieldSize])
	offset += paddingSizeFieldSize

	p.transferLength = byteOrder.Uint32(buf[offset : offset+transferLengthFieldSize])
	offset += transferLengthFieldSize

	p.blockID = byteOrder.Uint32(buf[offset : offset+blockIDFieldSize])
	offset += blockIDFieldSize

	p.bcastHeight = buf[offset]
	offset += bcastHeightSize

	p.block = buf[offset:]
	return nil
}
