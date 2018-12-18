package payload

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/wire/commands"
)

// MsgVerAck has no payload.
type MsgVerAck struct{}

// NewMsgVerAck returns a MsgVerAck struct.
func NewMsgVerAck() *MsgVerAck {
	return &MsgVerAck{}
}

// Encode implements payload interface.
func (m *MsgVerAck) Encode(w io.Writer) error {
	return nil
}

// Decode implements payload interface.
func (m *MsgVerAck) Decode(r io.Reader) error {
	return nil
}

// Command returns the command string associated with the verack message.
// Implements payload interface.
func (m *MsgVerAck) Command() commands.Cmd {
	return commands.VerAck
}
