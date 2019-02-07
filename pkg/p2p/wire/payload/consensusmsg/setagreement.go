package consensusmsg

import (
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// SetAgreement defines a setagreement message on the Dusk wire protocol.
type SetAgreement struct {
	BlockHash []byte
	VoteSet   []*Vote
}

// NewSetAgreement returns a SetAgreement struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewSetAgreement(hash []byte, voteSet []*Vote) (*SetAgreement, error) {
	if len(hash) != 32 {
		return nil, errors.New("wire: supplied block hash for set agreement payload is improper length")
	}

	return &SetAgreement{
		BlockHash: hash,
		VoteSet:   voteSet,
	}, nil
}

// Encode a SetAgreement struct and write to w.
// Implements Msg interface.
func (s *SetAgreement) Encode(w io.Writer) error {
	if err := encoding.Write256(w, s.BlockHash); err != nil {
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

// Decode a SetAgreement from r.
// Implements Msg interface.
func (s *SetAgreement) Decode(r io.Reader) error {
	if err := encoding.Read256(r, &s.BlockHash); err != nil {
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
func (s *SetAgreement) Type() ID {
	return SetAgreementID
}
