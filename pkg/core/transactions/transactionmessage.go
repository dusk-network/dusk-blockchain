package transactions

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
)

// TransactionMessage defines a Dusk wire message containing a transaction.
type TransactionMessage struct {
	Tx *Stealth
}

// NewTransactionMessage returns a TransactionMessage populated with the specified tx.
func NewTransactionMessage(tx *Stealth) *TransactionMessage {
	return &TransactionMessage{
		Tx: tx,
	}
}

// Encode a MsgTX struct and write to w.
// Implements payload interface.
func (m *TransactionMessage) Encode(w io.Writer) error {
	return m.Tx.Encode(w)
}

// Decode a MsgTX from r.
// Implements payload interface.
func (m *TransactionMessage) Decode(r io.Reader) error {
	m.Tx = &Stealth{}
	return m.Tx.Decode(r)
}

// Command returns the command string associated with the tx message.
// Implements payload interface.
func (m *TransactionMessage) Command() commands.Cmd {
	return commands.Tx
}
