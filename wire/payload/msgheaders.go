package payload

import (
	"gitlab.dusk.network/dusk-core/dusk-go/wire/commands"
	"io"
)

// MsgHeaders defines a headers message on the Dusk wire protocol.
type MsgHeaders struct {
}

// Finish this when block structure is defined

// Encode implements payload interface.
func (m *MsgHeaders) Encode(w io.Writer) error {
	// Implement when Block structure is known
	return nil
}

// Decode implements payload interface.
func (m *MsgHeaders) Decode(r io.Reader) error {
	// Implement when Block structure is known
	return nil
}

// Command returns the command string associated with the getaddr message.
// Implements payload interface.
func (m *MsgHeaders) Command() commands.Cmd {
	return commands.GetAddr
}
