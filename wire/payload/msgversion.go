package payload

import (
	"encoding/binary"
	"io"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/wire/commands"
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
func NewMsgVersion(version uint32, from *NetAddress, to *NetAddress) *MsgVersion {
	return &MsgVersion{
		Version:     version,
		Timestamp:   uint32(time.Now().Unix()),
		FromAddress: from,
		ToAddress:   to,
	}
}

// Encode a MsgVersion struct and write to w.
// Implements Payload interface.
func (m *MsgVersion) Encode(w io.Writer) error {
	if err := encoding.WriteUint32(w, binary.LittleEndian, m.Version); err != nil {
		return err
	}

	if err := encoding.WriteUint32(w, binary.LittleEndian, m.Timestamp); err != nil {
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
// Implements Payload interface.
func (m *MsgVersion) Decode(r io.Reader) error {
	if err := encoding.ReadUint32(r, binary.LittleEndian, &m.Version); err != nil {
		return err
	}

	if err := encoding.ReadUint32(r, binary.LittleEndian, &m.Timestamp); err != nil {
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

	m.FromAddress = &from
	m.ToAddress = &to
	return nil
}

// Command returns the command string associated with the version message.
// Implements Payload interface.
func (m *MsgVersion) Command() commands.Cmd {
	return commands.Version
}
