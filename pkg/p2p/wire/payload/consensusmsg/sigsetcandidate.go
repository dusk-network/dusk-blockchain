package consensusmsg

import (
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// SigSetCandidate defines a signatureset message on the Dusk wire protocol.
type SigSetCandidate struct {
	WinningBlockHash []byte  // Winning block hash
	SignatureSet     []*Vote // Generated signature set
	Step             uint8   // Step at which this vote set was created
}

// NewSigSetCandidate returns a SigSetCandidate struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewSigSetCandidate(winningBlock []byte, sigSet []*Vote, step uint8) (*SigSetCandidate, error) {
	if len(winningBlock) != 32 {
		return nil, errors.New("wire: supplied winning block hash for signature set candidate payload is improper length")
	}

	return &SigSetCandidate{
		WinningBlockHash: winningBlock,
		SignatureSet:     sigSet,
		Step:             step,
	}, nil
}

// Encode a SigSetCandidate struct and write to w.
// Implements Msg interface.
func (s *SigSetCandidate) Encode(w io.Writer) error {
	if err := encoding.Write256(w, s.WinningBlockHash); err != nil {
		return err
	}

	if err := encoding.WriteVarInt(w, uint64(len(s.SignatureSet))); err != nil {
		return err
	}

	for _, vote := range s.SignatureSet {
		if err := vote.Encode(w); err != nil {
			return err
		}
	}

	if err := encoding.WriteUint8(w, s.Step); err != nil {
		return err
	}

	return nil
}

// Decode a SigSetCandidate from r.
// Implements Msg interface.
func (s *SigSetCandidate) Decode(r io.Reader) error {
	if err := encoding.Read256(r, &s.WinningBlockHash); err != nil {
		return err
	}

	lVotes, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	s.SignatureSet = make([]*Vote, lVotes)
	for i := uint64(0); i < lVotes; i++ {
		s.SignatureSet[i] = &Vote{}
		if err := s.SignatureSet[i].Decode(r); err != nil {
			return err
		}
	}

	if err := encoding.ReadUint8(r, &s.Step); err != nil {
		return err
	}

	return nil
}

// Type returns the consensus payload identifier.
// Implements Msg interface.
func (s *SigSetCandidate) Type() ID {
	return SigSetCandidateID
}
