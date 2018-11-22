package payload

import (
	"io"

	"github.com/toghrulmaharramov/dusk-go/transactions"
	"github.com/toghrulmaharramov/dusk-go/wire/commands"
)

// MsgTx defines a Dusk wire message containing a transaction.
type MsgTx struct {
	Tx transactions.Stealth
}

// NewMsgTx returns a MsgTx populated with the specified tx.
func NewMsgTx(tx transactions.Stealth) *MsgTx {
	return &MsgTx{
		Tx: tx,
	}
}

// Encode a MsgTX struct and write to w.
// Implements payload interface.
func (m *MsgTx) Encode(w io.Writer) error {
	return m.Tx.Encode(w)
}

// Decode a MsgTX from r.
// Implements payload interface.
func (m *MsgTx) Decode(r io.Reader) error {
	return m.Tx.Decode(r)
}

// Command returns the command string associated with the tx message.
// Implements payload interface.
func (m *MsgTx) Command() commands.Cmd {
	return commands.Tx
}
