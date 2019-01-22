package consensusmsg

import "io"

// Candidate defines a candidate score consensus payload on the Dusk wire protocol.
type Candidate struct {
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
