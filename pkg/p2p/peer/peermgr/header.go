package peermgr

import (
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// MessageHeader defines a Dusk wire message header.
type MessageHeader struct {
	Magic    protocol.Magic // 4 bytes
	Topic    topics.Topic   // 15 bytes
	Length   uint32         // 4 bytes
	Checksum uint32         // 4 bytes
}

// MessageHeaderSize defines the size of a Dusk wire message header in bytes.
const MessageHeaderSize = 4 + topics.Size + 4 + 4

// decodeMessageHeader will decode a header from a Dusk wire message.
func decodeMessageHeader(r io.Reader) (*MessageHeader, error) {
	var magic uint32
	if err := encoding.ReadUint32(r, binary.LittleEndian, &magic); err != nil {
		return nil, err
	}

	var cmdBuf [topics.Size]byte
	if _, err := r.Read(cmdBuf[:]); err != nil {
		return nil, err
	}

	topic := topics.ByteArrayToTopic(cmdBuf)

	var length uint32
	if err := encoding.ReadUint32(r, binary.LittleEndian, &length); err != nil {
		return nil, err
	}

	var checksum uint32
	if err := encoding.ReadUint32(r, binary.LittleEndian, &checksum); err != nil {
		return nil, err
	}

	return &MessageHeader{
		Magic:    protocol.Magic(magic),
		Topic:    topic,
		Length:   length,
		Checksum: checksum,
	}, nil
}
