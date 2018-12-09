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
	Version     uint32
	Timestamp   uint32
	FromAddress *NetAddress
	ToAddress   *NetAddress
}

// NewMsgVersion returns a populated MsgVersion struct. The node's
// I2P address should be passed as an argument.
func NewMsgVersion(dest string, version uint32, from *NetAddress, to *NetAddress) *MsgVersion {
	return &MsgVersion{
		Version:     version,
		Timestamp:   uint32(time.Now().Unix()),
		FromAddress: from,
		ToAddress:   to,
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

	if err := m.FromAddress.Encode(w); err != nil {
		return err
	}

	if err := m.ToAddress.Encode(w); err != nil {
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

	var from NetAddress
	var to NetAddress
	if err := from.Decode(r); err != nil {
		return err
	}

	if err := to.Decode(r); err != nil {
		return err
	}

	m.Version = v
	m.Timestamp = t
	m.FromAddress = &from
	m.ToAddress = &to
	return nil
}

// Command returns the command string associated with the version message.
// Implements the Payload interface.
func (m *MsgVersion) Command() commands.Cmd {
	return commands.Version
}
