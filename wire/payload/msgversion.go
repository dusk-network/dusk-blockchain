package payload

import (
	"encoding/binary"
	"io"
	"time"

	"github.com/toghrulmaharramov/dusk-go/encoding"
	"github.com/toghrulmaharramov/dusk-go/wire/commands"
)

// MsgVersion declares a version message on the Dusk wire protocol.
// This is used by a node to advertise itself to peers on the network.
type MsgVersion struct {
	Version   uint32
	Timestamp uint32
	Dest      string
}

// NewMsgVersion returns a populated MsgVersion struct. The node's
// I2P address should be passed as an argument.
func NewMsgVersion(dest string) *MsgVersion {
	return &MsgVersion{
		Version:   1, // Get this from somewhere else in the future
		Timestamp: uint32(time.Now().Unix()),
		Dest:      dest, // Possibly make and use a getter function instead
	}
}

// Encode a MsgVersion struct and write to w.
// Implements the Payload interface.
func (m *MsgVersion) Encode(w io.Writer) error {
	if err := encoding.PutUint32(w, binary.LittleEndian, m.Version); err != nil {
		return err
	}

	if err := encoding.PutUint32(w, binary.LittleEndian, m.Timestamp); err != nil {
		return err
	}

	if err := encoding.WriteString(w, m.Dest); err != nil {
		return err
	}

	return nil
}

// Decode a MsgVersion from r.
// Implements the Payload interface
func (m *MsgVersion) Decode(r io.Reader) error {
	v, err := encoding.Uint32(r, binary.LittleEndian)
	if err != nil {
		return err
	}

	t, err := encoding.Uint32(r, binary.LittleEndian)
	if err != nil {
		return err
	}

	d, err := encoding.ReadString(r)
	if err != nil {
		return err
	}

	m.Version = v
	m.Timestamp = t
	m.Dest = d
	return nil
}

// Command returns the command string associated with the version message.
// Implements the Payload interface.
func (m *MsgVersion) Command() commands.Cmd {
	return commands.Version
}
