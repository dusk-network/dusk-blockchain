package wire

import (
	"bytes"
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

// Payload defines the message payload.
type Payload interface {
	Encode(w io.Writer) error
	Decode(r io.Reader) error
	Command() commands.Cmd
}

// WriteMessage will write a Dusk wire message to w.
func WriteMessage(w io.Writer, magic protocol.Magic, p Payload) error {
	if err := encoding.WriteUint32(w, binary.LittleEndian, uint32(magic)); err != nil {
		return err
	}

	byteCmd := commands.CmdToByteArray(p.Command())
	if err := binary.Write(w, binary.LittleEndian, byteCmd); err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if err := p.Encode(buf); err != nil {
		return err
	}

	payloadLength := uint32(buf.Len())
	checksum, err := crypto.Checksum(buf.Bytes())
	if err != nil {
		return err
	}

	if err := encoding.WriteUint32(w, binary.LittleEndian, payloadLength); err != nil {
		return err
	}

	if err := encoding.WriteUint32(w, binary.LittleEndian, checksum); err != nil {
		return err
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return err
	}

	return nil
}
