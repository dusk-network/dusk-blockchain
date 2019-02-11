package consensusmsg

import (
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// SigSetAgreement defines a sigsetagreement message on the Dusk wire protocol.
type SigSetAgreement struct {
	BlockHash []byte
	SetHash   []byte
	VoteSet   []*Vote
}

// NewSigSetAgreement returns a SigSetAgreement struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewSigSetAgreement(blockHash, setHash []byte, voteSet []*Vote) (*SigSetAgreement, error) {
	if len(blockHash) != 32 {
		return nil, errors.New("wire: supplied block hash for signature set agreement payload is improper length")
	}

	if len(setHash) != 32 {
		return nil, errors.New("wire: supplied set hash for signature set agreement payload is improper length")
	}

	return &SigSetAgreement{
		BlockHash: blockHash,
		SetHash:   setHash,
		VoteSet:   voteSet,
	}, nil
}

// Encode a SigSetAgreement struct and write to w.
// Implements Msg interface.
func (s *SigSetAgreement) Encode(w io.Writer) error {
	if err := encoding.Write256(w, s.BlockHash); err != nil {
		return err
	}

	if err := encoding.Write256(w, s.SetHash); err != nil {
		return err
	}

	if err := encoding.WriteVarInt(w, uint64(len(s.VoteSet))); err != nil {
		return err
	}

	for _, vote := range s.VoteSet {
		if err := vote.Encode(w); err != nil {
			return err
		}
	}

	return nil
}

// Decode a BlockAgreement from r.
// Implements Msg interface.
func (s *SigSetAgreement) Decode(r io.Reader) error {
	if err := encoding.Read256(r, &s.BlockHash); err != nil {
		return err
	}

	if err := encoding.Read256(r, &s.SetHash); err != nil {
		return err
	}

	lVotes, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	s.VoteSet = make([]*Vote, lVotes)
	for i := uint64(0); i < lVotes; i++ {
		s.VoteSet[i] = &Vote{}
		if err := s.VoteSet[i].Decode(r); err != nil {
			return err
		}
	}

	return nil
}

// Type returns the consensus payload identifier.
// Implements Msg interface.
func (s *SigSetAgreement) Type() ID {
	return SigSetAgreementID
}
