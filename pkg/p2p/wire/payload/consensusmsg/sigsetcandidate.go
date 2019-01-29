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
	PubKeyBLS        []byte  // Node BLS public key
	Score            []byte  // Node sortition score
}

// NewSigSetCandidate returns a SigSetCandidate struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewSigSetCandidate(winningBlock []byte, sigSet []*Vote, pubKeyBLS, score []byte) (*SigSetCandidate, error) {
	if len(winningBlock) != 32 {
		return nil, errors.New("wire: supplied winning block hash for signature set candidate payload is improper length")
	}

	if len(score) != 33 {
		return nil, errors.New("wire: supplied score for signature set candidate payload is improper length")
	}

	return &SigSetCandidate{
		WinningBlockHash: winningBlock,
		SignatureSet:     sigSet,
		PubKeyBLS:        pubKeyBLS,
		Score:            score,
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

	if err := encoding.WriteVarBytes(w, s.PubKeyBLS); err != nil {
		return err
	}

	if err := encoding.WriteBLS(w, s.Score); err != nil {
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

	if err := encoding.ReadVarBytes(r, &s.PubKeyBLS); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &s.Score); err != nil {
		return err
	}

	return nil
}

// Type returns the consensus payload identifier.
// Implements Msg interface.
func (s *SigSetCandidate) Type() ID {
	return SigSetCandidateID
}
