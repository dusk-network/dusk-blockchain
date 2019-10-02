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
	if uint64(buf.Len()) > MaxFrameSize {
		return fmt.Errorf("message size exceeds MaxFrameSize (%d)", MaxFrameSize)
	}

	msg := new(bytes.Buffer)
	// Append prefix(header)
	if err := encoding.WriteUint64LE(msg, uint64(buf.Len())); err != nil {
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

func ReadFrame(r io.Reader) ([]byte, error) {
	sizeBytes := make([]byte, 8)
	if _, err := io.ReadFull(r, sizeBytes); err != nil {
		return nil, err
	}

	size := binary.LittleEndian.Uint64(sizeBytes)
	if size > MaxFrameSize {
		return nil, fmt.Errorf("message size exceeds MaxFrameSize (%d), %d", MaxFrameSize, size)
	}

	buf := make([]byte, int(size))
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}

	return buf, nil
}
