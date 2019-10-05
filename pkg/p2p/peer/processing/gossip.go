package processing

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
)

type (
	headerWriter struct {
		magic protocol.Magic
	}

	// Gossip is a preprocessor for gossip messages.
	Gossip struct {
		magic protocol.Magic
	}
)

// NewGossip returns a gossip preprocessor with the specified magic.
func NewGossip(magic protocol.Magic) *Gossip {
	return &Gossip{
		magic: magic,
	}
}

// Process a message that is passing through, by prepending the network magic to the
// buffer, and then COBS encoding it.
func (g *Gossip) Process(m *bytes.Buffer) error {
	buf := g.magic.ToBuffer()
	if _, err := buf.ReadFrom(m); err != nil {
		return err
	}

	if err := WriteFrame(&buf); err != nil {
		return err
	}

	*m = buf
	return nil
}
