package block

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
)

// MsgBlock defines a block message on the Dusk wire protocol.
type MsgBlock struct {
	Block *block.Block
}

// NewMsgBlock will return a MsgBlock with the specified block
// as it's contents.
func NewMsgBlock(b *block.Block) *MsgBlock {
	return &MsgBlock{
		Block: b,
	}
}

// Encode a MsgBlock struct and write to w.
// Implements payload interface.
func (m *MsgBlock) Encode(w io.Writer) error {
	return m.Block.Encode(w)
}

// Decode a MsgBlock from r.
// Implements payload interface.
func (m *MsgBlock) Decode(r io.Reader) error {
	m.Block = &block.Block{}
	return m.Block.Decode(r)
}

// Command returns the command string associated with the MsgBlock message.
// Implements the Payload interface.
func (m *MsgBlock) Command() commands.Cmd {
	return commands.Block
}
