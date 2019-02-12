package payload

import (
	"encoding/binary"
	"io"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// MsgVersion declares a version message on the Dusk wire protocol.
// This is used by a node to advertise itself to peers on the network.
type MsgVersion struct {
	Version     uint32
	Timestamp   uint32
	FromAddress *NetAddress
	ToAddress   *NetAddress
	Services    protocol.ServiceFlag
	Nonce       uint64
}

// NewMsgVersion returns a populated MsgVersion struct. The node's
// I2P address should be passed as an argument.
func NewMsgVersion(version uint32, from, to *NetAddress, services protocol.ServiceFlag,
	nonce uint64) *MsgVersion {
	return &MsgVersion{
		Version:     version,
		Timestamp:   uint32(time.Now().Unix()),
		FromAddress: from,
		ToAddress:   to,
		Services:    services,
		Nonce:       nonce,
	}
}

// Encode a MsgVersion struct and write to w.
// Implements Payload interface.
func (m *MsgVersion) Encode(w io.Writer) error {
	if err := encoding.WriteUint32(w, binary.LittleEndian, uint32(m.Version)); err != nil {
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

	if err := encoding.WriteUint64(w, binary.LittleEndian, uint64(m.Services)); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, uint64(m.Nonce)); err != nil {
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

	var services uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &services); err != nil {
		return err
	}

	m.Services = protocol.ServiceFlag(services)

	if err := encoding.ReadUint64(r, binary.LittleEndian, &m.Nonce); err != nil {
		return err
	}

	return nil
}

// Command returns the command string associated with the version message.
// Implements Payload interface.
func (m *MsgVersion) Command() commands.Cmd {
	return commands.Version
}
