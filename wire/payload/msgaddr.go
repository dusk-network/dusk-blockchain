package payload

import (
	"io"

	"github.com/toghrulmaharramov/dusk-go/encoding"
	"github.com/toghrulmaharramov/dusk-go/wire/commands"
)

// MsgAddr defines a Dusk wire message containing an active
// node's I2P address.
type MsgAddr struct {
	// An I2P address; length can vary depending on key generation
	// algorhithm used.
	DestList []string
}

// NewMsgAddr returns an empty MsgAddr struct. This MsgAddr struct
// can then be populated with the AddAddr function, ideally by
// going down a peerlist.
func NewMsgAddr() *MsgAddr {
	return &MsgAddr{}
}

// AddAddr adds an I2P address to the MsgAddr's DestList.
func (m *MsgAddr) AddAddr(dest string) {
	m.DestList = append(m.DestList, dest)
}

// Encode an address message.
// Implements payload interface.
func (m *MsgAddr) Encode(w io.Writer) error {
	if err := encoding.WriteVarInt(w, uint64(len(m.DestList))); err != nil {
		return err
	}

	for _, dest := range m.DestList {
		if err := encoding.WriteString(w, dest); err != nil {
			return err
		}
	}

	return nil
}

// Decode an address message.
// Implements payload interface.
func (m *MsgAddr) Decode(r io.Reader) error {
	n, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	m.DestList = make([]string, n)
	for i := uint64(0); i < n; i++ {
		dest, err := encoding.ReadString(r)
		if err != nil {
			return err
		}

		m.DestList[i] = dest
	}

	return nil
}

// Command returns the command string associated with the addr message.
// Implements payload interface.
func (m *MsgAddr) Command() commands.Cmd {
	return commands.Addr
}
