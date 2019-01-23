package consensusmsg

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
)

// Candidate defines a candidate score consensus payload on the Dusk wire protocol.
type Candidate struct {
	Block *block.Block
}

// Encode a Candidate struct to w
// Implements Msg interface.
func (c *Candidate) Encode(w io.Writer) error {
	return nil
}

// Decode a Candidate from r
// Implements Msg interface.
func (c *Candidate) Decode(r io.Reader) error {
	return nil
}

// Type returns the consensus payload identifier.
// Implements Msg interface.
func (c *Candidate) Type() ID {
	return CandidateID
}
