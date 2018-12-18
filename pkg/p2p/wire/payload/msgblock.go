package payload

import (
	"gitlab.dusk.network/dusk-core/dusk-go/wire/commands"
	"io"
)

// MsgBlock defines a block message on the Dusk wire protocol.
type MsgBlock struct {
	BlockHeader []byte
	// TODO: Block
}

// Finish this when block structure is defined.

// Encode a MsgBlock struct and write to w.
// Implements payload interface.
func (m *MsgBlock) Encode(w io.Writer) error {
	// Implement when Block structure is known
	return nil
}

// Decode a MsgBlock from r.
// Implements payload interface.
func (m *MsgBlock) Decode(r io.Reader) error {
	// Implement when Block structure is known
	return nil
}

// Command returns the command string associated with the MsgBlock message.
// Implements the Payload interface.
func (m *MsgBlock) Command() commands.Cmd {
	return commands.GetBlocks
}
