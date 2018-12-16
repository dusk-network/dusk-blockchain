package wire

import (
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/wire/commands"
)

// Header defines a Dusk wire message header.
type Header struct {
	Magic    uint32       // 4 bytes
	Command  commands.Cmd // 12 bytes
	Length   uint32       // 4 bytes
	Checksum uint32       // 4 bytes
}

// HeaderSize defines the size of a Dusk wire message header in bytes.
const HeaderSize = 4 + 12 + 4 + 4

// Decode will decode a header from a Dusk wire message.
func (h *Header) Decode(r io.Reader) error {
	magic, err := encoding.Uint32(r, binary.LittleEndian)
	if err != nil {
		return err
	}

	var buf [12]byte
	if _, err := r.Read(buf[:]); err != nil {
		return err
	}

	cmd := commands.ByteArrayToCmd(buf)
	length, err := encoding.Uint32(r, binary.LittleEndian)
	if err != nil {
		return err
	}

	checksum, err := encoding.Uint32(r, binary.LittleEndian)
	if err != nil {
		return err
	}

	h.Magic = magic
	h.Command = cmd
	h.Length = length
	h.Checksum = checksum
	return nil
}
