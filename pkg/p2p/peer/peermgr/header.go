package peermgr

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// MessageHeader defines a Dusk wire message header.
type MessageHeader struct {
	Magic    protocol.Magic // 4 bytes
	Command  topics.Topic   // 15 bytes
	Length   uint32         // 4 bytes
	Checksum uint32         // 4 bytes
}

// MessageHeaderSize defines the size of a Dusk wire message header in bytes.
const MessageHeaderSize = 4 + topics.Size + 4 + 4

// decodeMessageHeader will decode a header from a Dusk wire message.
func decodeMessageHeader(r io.Reader) (*MessageHeader, error) {
	// var magic uint32
	// if err := encoding.ReadUint32(r, binary.LittleEndian, &magic); err != nil {
	// 	return nil, err
	// }
	// h.Magic = protocol.Magic(magic)

	// var cmdBuf [topics.Size]byte
	// if _, err := r.Read(cmdBuf[:]); err != nil {
	// 	return nil, err
	// }

	// cmd := topics.ByteArrayToTopic(cmdBuf)
	// h.Command = cmd
	// if err := encoding.ReadUint32(r, binary.LittleEndian, &h.Length); err != nil {
	// 	return nil, err
	// }

	// if err := encoding.ReadUint32(r, binary.LittleEndian, &h.Checksum); err != nil {
	// 	return nil, err
	// }

	return nil, nil
}
