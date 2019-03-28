package peer

import (
	"bytes"
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
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

func addHeader(message *bytes.Buffer, magic protocol.Magic, topic topics.Topic) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	if err := encoding.WriteUint32(buffer, binary.LittleEndian, uint32(magic)); err != nil {
		return nil, err
	}

	if err := writeTopic(buffer, topic); err != nil {
		return nil, err
	}

	payloadLength := uint32(message.Len())
	if err := encoding.WriteUint32(buffer, binary.LittleEndian, payloadLength); err != nil {
		return nil, err
	}

	checksum, err := crypto.Checksum(message.Bytes())
	if err != nil {
		return nil, err
	}
	if err := encoding.WriteUint32(buffer, binary.LittleEndian, checksum); err != nil {
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
