// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/checksum"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

const (
	// MaxFrameSize is set at 1375000 bytes.
	MaxFrameSize = uint64(1375000)

	// reservedFieldSize is number of bytes the reserved field uses.
	reservedFieldSize = 8

	// VersionLength is number of bytes the version uses.
	VersionLength = uint64(8)
)

// WriteFrame same as WriteFrameWithReserved but with reserved field
// fixed to 0.
func WriteFrame(buf *bytes.Buffer, cs []byte) error {
	return WriteFrameWithReserved(buf, cs, 0)
}

// WriteFrameWithReserved mutates a buffer by adding a length-prefixing wire message frame at the beginning of the message.
func WriteFrameWithReserved(buf *bytes.Buffer, cs []byte, reserved uint64) error {
	ln := VersionLength + uint64(reservedFieldSize+checksum.Length+buf.Len())
	if ln > MaxFrameSize {
		return fmt.Errorf("message size exceeds MaxFrameSize (%d)", MaxFrameSize)
	}

	msg := new(bytes.Buffer)
	// Add length bytes
	if err := encoding.WriteUint64LE(msg, ln); err != nil {
		return err
	}

	// Add version
	vBuf := VersionAsBuffer()
	if _, err := msg.Write(vBuf.Bytes()); err != nil {
		return err
	}

	// Add reserved field bytes
	if err := encoding.WriteUint64LE(msg, reserved); err != nil {
		return err
	}

	// Add checksum
	if _, err := msg.Write(cs); err != nil {
		return err
	}

	// Append payload
	_, err := buf.WriteTo(msg)
	if err != nil {
		return err
	}

	*buf = *msg

	return nil
}

// ReadFrame extract the bytes representing the size of the packet and thus
// read the amount of bytes specified by such prefix in little endianness.
func ReadFrame(r io.Reader) (uint64, error) {
	var length uint64

	sizeBytes := make([]byte, 8)
	// This is used mainly for net.Conn, therefore io.ReadFull prevents weird
	// unbuffered reads which would terminate the reading operation before
	// actually reading 8 bytes.
	if _, err := io.ReadFull(r, sizeBytes); err != nil {
		return length, err
	}

	length = binary.LittleEndian.Uint64(sizeBytes)
	if length > MaxFrameSize {
		return 0, fmt.Errorf("message size exceeds MaxFrameSize (%d), %d", MaxFrameSize, length)
	}

	return length, nil
}
