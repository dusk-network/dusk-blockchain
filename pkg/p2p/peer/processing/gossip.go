package processing

import (
	"bytes"
	"errors"
	"io"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"golang.org/x/crypto/sha3"
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

// Process a message that is passing through, by prepending the protocol
// magic and the message checksum, and finally by prepending the length.
func (g *Gossip) Process(m *bytes.Buffer) error {
	digest := sha3.Sum256(m.Bytes())
	return WriteFrame(m, g.Magic, digest[0:ChecksumLength])
}

// UnpackLength unwraps the incoming packet (likely from a net.Conn struct) and returns the length of the packet without reading the payload (which is left to the user of this method)
func (g *Gossip) UnpackLength(r io.Reader) (uint64, error) {
	packetLength, err := ReadFrame(r)
	if err != nil {
		return 0, err
	}

	magic, err := protocol.Extract(r)
	if err != nil {
		return 0, err
	}

	if magic != g.Magic {
		return 0, errors.New("magic mismatch")
	}

	return packetLength - uint64(magic.Len()), nil
}
