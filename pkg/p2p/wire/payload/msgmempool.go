package payload

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
)

// MsgMemPool has no payload.
type MsgMemPool struct {
}

// NewMsgMemPool returns a MsgMemPool struct.
func NewMsgMemPool() *MsgMemPool {
	return &MsgMemPool{}
}

// Encode implements payload interface.
func (m *MsgMemPool) Encode(w io.Writer) error {
	return nil
}

// Decode implements payload interface.
func (m *MsgMemPool) Decode(r io.Reader) error {
	return nil
}

// Command returns the command string associated with the verack message.
// Implements payload interface.
func (m *MsgMemPool) Command() commands.Cmd {
	return commands.MemPool
}
