package processing

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
)

type (
	headerWriter struct {
		magicBuf bytes.Buffer
	}

	// Gossip is a preprocessor for gossip messages.
	Gossip struct {
		headerWriter
	}
)

func (h headerWriter) Write(m *bytes.Buffer) error {
	b := h.magicBuf

	if _, err := m.WriteTo(&b); err != nil {
		return err
	}

	*m = b
	return nil
}

// NewGossip returns a gossip preprocessor with the specified magic.
func NewGossip(magic protocol.Magic) *Gossip {
	return &Gossip{
		headerWriter: headerWriter{
			magicBuf: writeMagic(magic),
		},
	}
}

func writeMagic(magic protocol.Magic) bytes.Buffer {
	b := new(bytes.Buffer)
	_ = encoding.WriteUint32LE(b, uint32(magic))
	return *b
}

// Process a message that is passing through, by prepending the network magic to the
// buffer, and then COBS encoding it.
func (g *Gossip) Process(m *bytes.Buffer) error {
	if err := g.headerWriter.Write(m); err != nil {
		return err
	}

	return WriteFrame(m)
}
