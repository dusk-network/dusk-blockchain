package messages

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
)

type BlockMessage struct {
	Block *block.Block
}

func (m *BlockMessage) Encode(w io.Writer) error {
	return m.Block.Encode(w)
}

func (m *BlockMessage) Decode(r io.Reader) error {
	m.Block = &block.Block{}
	return m.Block.Decode(r)
}
