package payload

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
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
	return nil
}

// Decode a MsgTX from r.
// Implements payload interface.
func (m *MsgTx) Decode(r io.Reader) error {
	return nil
}

// Command returns the command string associated with the tx message.
// Implements payload interface.
func (m *MsgTx) Command() commands.Cmd {
	return commands.Tx
}
