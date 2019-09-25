package processing

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
)

type (
	headerWriter struct {
		magic protocol.Magic
	}

	// Gossip is a preprocessor for gossip messages.
	Gossip struct {
		headerWriter
	}
)

func (h *headerWriter) Write(m *bytes.Buffer) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := encoding.WriteUint32LE(buf, uint32(h.magic)); err != nil {
		return nil, err
	}

	if _, err := m.WriteTo(buf); err != nil {
		return nil, err
	}

	return buf, nil
}

// NewGossip returns a gossip preprocessor with the specified magic.
func NewGossip(magic protocol.Magic) *Gossip {
	return &Gossip{
		headerWriter: headerWriter{
			magic: magic,
		},
	}
}

// Process a message that is passing through, by prepending the network magic to the
// buffer, and then COBS encoding it.
func (g *Gossip) Process(m *bytes.Buffer) (*bytes.Buffer, error) {
	buf, err := g.headerWriter.Write(m)
	if err != nil {
		return nil, err
	}

	return WriteFrame(buf)
}
