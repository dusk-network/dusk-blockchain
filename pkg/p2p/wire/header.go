package wire

import (
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

// Header defines a Dusk wire message header.
type Header struct {
	Magic    protocol.DuskNetwork // 4 bytes
	Command  commands.Cmd         // 14 bytes
	Length   uint32               // 4 bytes
	Checksum uint32               // 4 bytes
}

// HeaderSize defines the size of a Dusk wire message header in bytes.
const HeaderSize = 4 + commands.Size + 4 + 4

// Decode will decode a header from a Dusk wire message.
func (h *Header) Decode(r io.Reader) error {
	var magic uint32
	if err := encoding.ReadUint32(r, binary.LittleEndian, &magic); err != nil {
		return err
	}
	h.Magic = protocol.DuskNetwork(magic)

	var cmdBuf [commands.Size]byte
	if _, err := r.Read(cmdBuf[:]); err != nil {
		return err
	}

	cmd := commands.ByteArrayToCmd(cmdBuf)
	h.Command = cmd
	if err := encoding.ReadUint32(r, binary.LittleEndian, &h.Length); err != nil {
		return err
	}

	if err := encoding.ReadUint32(r, binary.LittleEndian, &h.Checksum); err != nil {
		return err
	}

	return nil
}
