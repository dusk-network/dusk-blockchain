package consensusmsg

import (
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// SigSetCandidate defines a signatureset message on the Dusk wire protocol.
type SigSetCandidate struct {
	WinningBlockHash []byte // Winning block hash
	SignatureSet     []byte // Generated signature set
}

// NewSigSetCandidate returns a SigSetCandidate struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewSigSetCandidate(winningBlock, sigSet []byte) (*SigSetCandidate, error) {
	if len(winningBlock) != 32 {
		return nil, errors.New("wire: supplied winning block hash for signature set candidate payload is improper length")
	}

	return &SigSetCandidate{
		WinningBlockHash: winningBlock,
		SignatureSet:     sigSet,
	}, nil
}

// Encode a SigSetCandidate struct and write to w.
// Implements Msg interface.
func (s *SigSetCandidate) Encode(w io.Writer) error {
	if err := encoding.Write256(w, s.WinningBlockHash); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, s.SignatureSet); err != nil {
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

	if err := encoding.ReadVarBytes(r, &s.SignatureSet); err != nil {
		return err
	}

	return nil
}

// Type returns the consensus payload identifier.
// Implements Msg interface.
func (s *SigSetCandidate) Type() ID {
	return SigSetCandidateID
}
