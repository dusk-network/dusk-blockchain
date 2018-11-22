package payload

import (
	"io"

	"github.com/toghrulmaharramov/dusk-go/wire/commands"
)

// MsgPing has no payload.
type MsgPing struct{}

// NewMsgPing returns a MsgPing struct.
func NewMsgPing() *MsgPing {
	return &MsgPing{}
}

// Encode implements payload interface.
func (m *MsgPing) Encode(w io.Writer) error {
	return nil
}

// Decode implements payload interface.
func (m *MsgPing) Decode(r io.Reader) error {
	return nil
}

// Command returns the command string associated with the ping message.
// Implements payload interface.
func (m *MsgPing) Command() commands.Cmd {
	return commands.Ping
}
