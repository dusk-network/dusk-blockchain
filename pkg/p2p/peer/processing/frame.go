package processing

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

const (
	MaxFrameSize = uint64(250000)
)

//WriteFrame mutates a buffer by adding a length-prefixing wire message frame at the beginning of the message
func WriteFrame(buf *bytes.Buffer) error {
	ln := uint64(buf.Len())
	if ln > MaxFrameSize {
		return fmt.Errorf("message size exceeds MaxFrameSize (%d)", MaxFrameSize)
	}

	msg := new(bytes.Buffer)
	// Append prefix(header)
	if err := encoding.WriteUint64LE(msg, ln); err != nil {
		return err
	}

	// Append payload
	_, err := msg.ReadFrom(buf)
	if err != nil {
		return err
	}

	*buf = *msg
	// TODO: Append Checksum
	return nil
}

func ReadFrame(r io.Reader) (uint64, error) {
	var length uint64
	sizeBytes := make([]byte, 8)
	// this is used mainly for net.Conn, therefore io.ReadFull prevents weird
	// unbuffered reads which would terminate the reading operation before
	// actually reading 8 bytes
	if _, err := io.ReadFull(r, sizeBytes); err != nil {
		return length, err
	}

	length = binary.LittleEndian.Uint64(sizeBytes)
	if length > MaxFrameSize {
		return 0, fmt.Errorf("message size exceeds MaxFrameSize (%d), %d", MaxFrameSize, length)
	}

	return length, nil
}
