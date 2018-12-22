package payload

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// MsgGetBlocks defines a getblocks message on the Dusk wire protocol.
type MsgGetBlocks struct {
	Locator  []byte
	HashStop []byte
}

// NewMsgGetBlocks returns a MsgGetBlocks struct with the specified
// locator and stop hash.
func NewMsgGetBlocks(locator []byte, stop []byte) *MsgGetBlocks {
	return &MsgGetBlocks{
		Locator:  locator,
		HashStop: stop,
	}
}

// Encode a MsgGetBlocks struct and write to w.
// Implements payload interface.
func (m *MsgGetBlocks) Encode(w io.Writer) error {
	if err := encoding.Write256(w, m.Locator); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.HashStop); err != nil {
		return err
	}

	return nil
}

// Decode a MsgGetBlocks from r.
// Implements payload interface.
func (m *MsgGetBlocks) Decode(r io.Reader) error {
	if err := encoding.Read256(r, &m.Locator); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.HashStop); err != nil {
		return err
	}

	return nil
}

// Command returns the command string associated with the getblocks message.
// Implements the Payload interface.
func (m *MsgGetBlocks) Command() commands.Cmd {
	return commands.GetBlocks
}
