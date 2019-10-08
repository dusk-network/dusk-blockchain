package processing

import (
	"bytes"
	"errors"
	"io"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
)

type (
	// Gossip is a preprocessor for gossip messages.
	Gossip struct {
		Magic protocol.Magic
	}
)

// NewGossip returns a gossip preprocessor with the specified magic.
func NewGossip(magic protocol.Magic) *Gossip {
	return &Gossip{
		Magic: magic,
	}
}

// Process a message that is passing through, by prepending the network magic to the
// buffer, and then COBS encoding it.
func (g *Gossip) Process(m *bytes.Buffer) error {
	buf := g.Magic.ToBuffer()
	if _, err := buf.ReadFrom(m); err != nil {
		return err
	}

	if err := WriteFrame(&buf); err != nil {
		return err
	}

	*m = buf
	return nil
}

// UnpackLength unwraps the incoming packet (likely from a net.Conn struct) and returns the length of the packet without reading the payload (which is left to the user of this method)
func (g *Gossip) UnpackLength(r io.Reader) (uint64, error) {
	var magic protocol.Magic
	packetLength, err := ReadFrame(r)
	if err != nil {
		return 0, err
	}

	magic, err = protocol.Extract(r)
	if err != nil {
		return 0, err
	}

	if magic != g.Magic {
		return 0, errors.New("magic mismatch")
	}

	return packetLength - uint64(magic.Len()), nil
}
