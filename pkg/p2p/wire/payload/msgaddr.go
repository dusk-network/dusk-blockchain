package payload

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// MsgAddr defines a Dusk wire message containing IP addresses
// of active nodes.
type MsgAddr struct {
	// A list of IP addresses,
	Addresses []*NetAddress
}

// NewMsgAddr returns an empty MsgAddr struct. This MsgAddr struct
// can then be populated with the AddAddr function, ideally by
// going down a peerlist.
func NewMsgAddr() *MsgAddr {
	return &MsgAddr{}
}

// AddAddr adds an I2P address to the MsgAddr's DestList.
func (m *MsgAddr) AddAddr(addr *NetAddress) {
	m.Addresses = append(m.Addresses, addr)
}

// Encode a MsgAddr struct and write to w.
// Implements payload interface.
func (m *MsgAddr) Encode(w io.Writer) error {
	if err := encoding.WriteVarInt(w, uint64(len(m.Addresses))); err != nil {
		return err
	}

	for _, dest := range m.Addresses {
		if err := dest.Encode(w); err != nil {
			return err
		}
	}

	return nil
}

// Decode a MsgAddr from r.
// Implements payload interface.
func (m *MsgAddr) Decode(r io.Reader) error {
	n, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	m.Addresses = make([]*NetAddress, n)
	for i := uint64(0); i < n; i++ {
		var addr NetAddress
		if err := addr.Decode(r); err != nil {
			return err
		}

		m.Addresses[i] = &addr
	}

	return nil
}

// Command returns the command string associated with the addr message.
// Implements payload interface.
func (m *MsgAddr) Command() commands.Cmd {
	return commands.Addr
}
