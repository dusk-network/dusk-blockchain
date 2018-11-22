package payload

import (
	"io"

	"github.com/toghrulmaharramov/dusk-go/transactions"
	"github.com/toghrulmaharramov/dusk-go/wire/commands"
)

// MsgTX defines a Dusk wire message containing a transaction.
type MsgTX struct {
	TX transactions.Stealth
}

// NewMsgTX returns a MsgTX populated with the specified tx.
func NewMsgTX(tx transactions.Stealth) *MsgTX {
	return &MsgTX{
		TX: tx,
	}
}

// Encode a MsgTX and write to w.
// Implements payload interface.
func (m *MsgTX) Encode(w io.Writer) error {
	return m.TX.Encode(w)
}

// Decode a MsgTX from r.
// Implements payload interface.
func (m *MsgTX) Decode(r io.Reader) error {
	return m.TX.Decode(r)
}

// Command returns the command string associated with the tx message.
// Implements payload interface.
func (m *MsgTX) Command() commands.Cmd {
	return commands.TX
}
