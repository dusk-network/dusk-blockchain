// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/checksum"
)

type (
	// Gossip is a preprocessor/reader for gossip messages.
	Gossip struct {
	}
)

// NewGossip returns a gossip preprocessor with the specified magic.
func NewGossip() *Gossip {
	return &Gossip{}
}

// Process same as ProcessWithReserved but with reserved field fixed to 0.
func (g *Gossip) Process(m *bytes.Buffer) error {
	return g.ProcessWithReserved(m, 0)
}

// ProcessWithReserved process a message that is passing through, by prepending
// magic and the message checksum, and finally by prepending the length. It
// allows to set reserved field.
func (g *Gossip) ProcessWithReserved(m *bytes.Buffer, reserved uint64) error {
	cs := checksum.Generate(m.Bytes())
	return WriteFrameWithReserved(m, cs, reserved)
}

// UnpackLength unwraps the incoming packet (likely from a net.Conn struct) and returns the length of the packet without reading the payload (which is left to the user of this method).
func (g *Gossip) UnpackLength(r io.Reader) (uint64, error) {
	packetLength, err := ReadFrame(r)
	if err != nil {
		return 0, err
	}

	version, err := ExtractVersion(r)
	if err != nil {
		return 0, err
	}

	if !VersionConstraint.Check(version) {
		return 0, fmt.Errorf("invalid message version %s received, expected %s", version, VersionConstraintString)
	}

	// if magic != g.Magic {
	// 	return 0, fmt.Errorf("magic mismatch, received %s expected %s", magic, g.Magic)
	// }

	// Reserved field is the message timestamp in DevNet/TestNet
	_, rfSize, err := g.extractReservedField(r)
	if err != nil {
		return 0, errors.New("reserved field mismatch")
	}

	// Uncomment on measuring average arrival time
	// s.registerPacket(packetLength, reservedFieldValue)

	// If packetLength is less than magic.Len(), ln is close to MaxUint64
	// due to integer overflow
	ln := packetLength - uint64(rfSize) - VersionLength

	if ln > MaxFrameSize {
		return 0, fmt.Errorf("invalid packet length %d", packetLength)
	}

	return ln, nil
}

// ReadMessage reads from the connection.
// TODO: Replace ReadMessage with ReadFrame.
func (g *Gossip) ReadMessage(src io.Reader) ([]byte, error) {
	length, err := g.UnpackLength(src)
	if err != nil {
		return nil, err
	}

	// read a [length]byte from connection
	buf := make([]byte, int(length))

	_, err = io.ReadFull(src, buf)
	if err != nil {
		return nil, err
	}

	return buf, err
}

// ReadFrame extract message from gossip frame, if no errors found.
func (g *Gossip) ReadFrame(src io.Reader) ([]byte, error) {
	length, err := g.UnpackLength(src)
	if err != nil {
		return nil, err
	}

	// read a [length]byte from connection
	buf := make([]byte, int(length))

	_, err = io.ReadFull(src, buf)
	if err != nil {
		return nil, err
	}

	message, cs, err := checksum.Extract(buf)
	if err != nil {
		return nil, fmt.Errorf("extracting checksum: %s", err.Error())
	}

	if !checksum.Verify(message, cs) {
		return nil, fmt.Errorf("invalid checksum: %s", err.Error())
	}

	return message, nil
}

func (g *Gossip) extractReservedField(r io.Reader) (int64, int, error) {
	size := 8
	// Read timestamp
	buffer := make([]byte, size)
	if _, err := io.ReadFull(r, buffer); err != nil {
		return 0, 0, err
	}

	return int64(binary.LittleEndian.Uint64(buffer)), size, nil
}
