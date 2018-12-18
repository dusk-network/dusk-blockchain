package payload

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/wire/commands"
)

// MsgGetHeaders defines a getheaders message on the Dusk wire protocol.
type MsgGetHeaders struct {
	Locator  []byte
	HashStop []byte
}

// NewMsgGetHeaders returns a MsgGetHeaders struct with the specified
// locator and stop hash.
func NewMsgGetHeaders(locator []byte, stop []byte) *MsgGetHeaders {
	return &MsgGetHeaders{
		Locator:  locator,
		HashStop: stop,
	}
}

// Encode a MsgGetHeaders struct and write to w.
// Implements payload interface.
func (m *MsgGetHeaders) Encode(w io.Writer) error {
	if err := encoding.Write256(w, m.Locator); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.HashStop); err != nil {
		return err
	}

	return nil
}

// Decode a MsgGetHeaders from r.
// Implements payload interface.
func (m *MsgGetHeaders) Decode(r io.Reader) error {
	if err := encoding.Read256(r, &m.Locator); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.HashStop); err != nil {
		return err
	}

	return nil
}

// Command returns the command string associated with the GetHeaders message.
// Implements the Payload interface.
func (m *MsgGetHeaders) Command() commands.Cmd {
	return commands.GetHeaders
}
