package processing

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

const (
	MaxFrameSize = uint64(250000)
)

//BuildFrame builds a length-prefixing wire message frame
func WriteFrame(buf *bytes.Buffer) (*bytes.Buffer, error) {
	if uint64(buf.Len()) > MaxFrameSize {
		return nil, fmt.Errorf("message size exceeds MaxFrameSize (%d)", MaxFrameSize)
	}

	msg := new(bytes.Buffer)
	// Append prefix(header)
	if err := encoding.WriteUint64(msg, binary.LittleEndian, uint64(buf.Len())); err != nil {
		return nil, err
	}

	// Append payload
	_, err := msg.ReadFrom(buf)
	if err != nil {
		return nil, err
	}

	// TODO: Append Checksum

	return msg, nil
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
