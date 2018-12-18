package payload

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/wire/commands"
)

// MsgGetAddr has no payload.
type MsgGetAddr struct{}

// NewMsgGetAddr returns a MsgGetAddr struct.
func NewMsgGetAddr() *MsgGetAddr {
	return &MsgGetAddr{}
}

// Encode implements payload interface.
func (m *MsgGetAddr) Encode(w io.Writer) error {
	return nil
}

// Decode implements payload interface.
func (m *MsgGetAddr) Decode(r io.Reader) error {
	return nil
}

// Command returns the command string associated with the getaddr message.
// Implements payload interface.
func (m *MsgGetAddr) Command() commands.Cmd {
	return commands.GetAddr
}
