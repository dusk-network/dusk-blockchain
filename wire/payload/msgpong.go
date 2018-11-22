package payload

import (
	"io"

	"github.com/toghrulmaharramov/dusk-go/wire/commands"
)

// MsgPong has no payload.
type MsgPong struct{}

// NewMsgPong returns a MsgPong struct.
func NewMsgPong() *MsgPong {
	return &MsgPong{}
}

// Encode implements payload interface.
func (m *MsgPong) Encode(w io.Writer) error {
	return nil
}

// Decode implements payload interface.
func (m *MsgPong) Decode(r io.Reader) error {
	return nil
}

// Command returns the command string associated with the pong message.
// Implements payload interface.
func (m *MsgPong) Command() commands.Cmd {
	return commands.Pong
}
