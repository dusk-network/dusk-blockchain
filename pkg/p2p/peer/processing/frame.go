package processing

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/checksum"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
)

const (
	MaxFrameSize = uint64(250000)
)

//WriteFrame mutates a buffer by adding a length-prefixing wire message frame at the beginning of the message
func WriteFrame(buf *bytes.Buffer, magic protocol.Magic, cs []byte) error {
	ln := uint64(magic.Len() + checksum.Length + buf.Len())
	if ln > MaxFrameSize {
		return fmt.Errorf("message size exceeds MaxFrameSize (%d)", MaxFrameSize)
	}

	msg := new(bytes.Buffer)
	// Add length bytes
	if err := encoding.WriteUint64LE(msg, ln); err != nil {
		return err
	}

	// Add magic
	mBuf := magic.ToBuffer()
	if _, err := msg.Write(mBuf.Bytes()); err != nil {
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
