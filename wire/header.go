package wire

import (
	"encoding/binary"
	"io"

	"github.com/toghrulmaharramov/dusk-go/encoding"
	"github.com/toghrulmaharramov/dusk-go/wire/commands"
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

	buf := make([]byte, commands.CommandSize)
	if _, err := r.Read(buf); err != nil {
		return err
	}

	cmd := commands.Cmd(buf)
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
