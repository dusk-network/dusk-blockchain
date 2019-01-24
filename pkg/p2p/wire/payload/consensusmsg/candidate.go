package consensusmsg

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
)

// Candidate defines a candidate consensus payload on the Dusk wire protocol.
type Candidate struct {
	Block *block.Block
}

// NewCandidate returns a candidate consensus payload with the specified block.
func NewCandidate(blk *block.Block) *Candidate {
	return &Candidate{
		Block: blk,
	}
}

// Encode a Candidate struct to w
// Implements Msg interface.
func (c *Candidate) Encode(w io.Writer) error {
	return c.Block.Encode(w)
}

// Decode a Candidate from r
// Implements Msg interface.
func (c *Candidate) Decode(r io.Reader) error {
	c.Block = &block.Block{}
	return c.Block.Decode(r)
}

// Type returns the consensus payload identifier.
// Implements Msg interface.
func (c *Candidate) Type() ID {
	return CandidateID
}
