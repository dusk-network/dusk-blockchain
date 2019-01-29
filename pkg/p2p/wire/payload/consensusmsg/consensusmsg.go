package consensusmsg

import "io"

// ID is an identifier for a consensus payload
type ID uint8

// Consensus identifiers
var (
	CandidateScoreID  ID = 0x00
	CandidateID       ID = 0x01
	ReductionID       ID = 0x02
	SetAgreementID    ID = 0x03
	SigSetCandidateID ID = 0x04
	SigSetVoteID      ID = 0x05
)

// Msg is an interface for consensus message payloads.
type Msg interface {
	Encode(w io.Writer) error
	Decode(r io.Reader) error
	Type() ID
}
