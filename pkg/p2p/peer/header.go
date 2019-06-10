package peer

import (
	"bytes"
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// MessageHeader defines a Dusk wire message header.
type MessageHeader struct {
	Magic  protocol.Magic // 4 bytes
	Length uint32         // 4 bytes
	Topic  topics.Topic   // 15 bytes
}

// MessageHeaderSize defines the size of a Dusk wire message header in bytes.
const MessageHeaderSize = 4 + 4

// decodeMessageHeader will decode a header from a Dusk wire message.
func decodeMessageHeader(r io.Reader) (*MessageHeader, error) {
	var magic uint32
	if err := encoding.ReadUint32(r, binary.LittleEndian, &magic); err != nil {
		return nil, err
	}

	var length uint32
	if err := encoding.ReadUint32(r, binary.LittleEndian, &length); err != nil {
		return nil, err
	}

	return &MessageHeader{
		Magic:  protocol.Magic(magic),
		Length: length,
		Topic:  topics.Version,
	}, nil
}

func addHeader(message *bytes.Buffer, magic protocol.Magic, topic topics.Topic) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	if err := encoding.WriteUint32(buffer, binary.LittleEndian, uint32(magic)); err != nil {
		return nil, err
	}

	payloadLength := uint32(message.Len())
	if err := encoding.WriteUint32(buffer, binary.LittleEndian, payloadLength); err != nil {
		return nil, err
	}

	if _, err := buffer.Write(message.Bytes()); err != nil {
		return nil, err
	}

	return buffer, nil
}

func writeTopic(r *bytes.Buffer, topic topics.Topic) error {
	topicBytes := topics.TopicToByteArray(topic)
	if _, err := r.Write(topicBytes[:]); err != nil {
		return err
	}

	return nil
}
