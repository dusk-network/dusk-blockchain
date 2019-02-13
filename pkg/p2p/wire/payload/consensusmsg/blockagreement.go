package consensusmsg

import (
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// BlockAgreement defines a blockagreement message on the Dusk wire protocol.
type BlockAgreement struct {
	BlockHash []byte
	VoteSet   []*Vote
}

// NewBlockAgreement returns a BlockAgreement struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewBlockAgreement(hash []byte, voteSet []*Vote) (*BlockAgreement, error) {
	if len(hash) != 32 {
		return nil, errors.New("wire: supplied block hash for block agreement payload is improper length")
	}

	return &BlockAgreement{
		BlockHash: hash,
		VoteSet:   voteSet,
	}, nil
}

// Encode a BlockAgreement struct and write to w.
// Implements Msg interface.
func (b *BlockAgreement) Encode(w io.Writer) error {
	if err := encoding.Write256(w, b.BlockHash); err != nil {
		return err
	}

	if err := encoding.WriteVarInt(w, uint64(len(b.VoteSet))); err != nil {
		return err
	}

	for _, vote := range b.VoteSet {
		if err := vote.Encode(w); err != nil {
			return err
		}
	}

	return nil
}

// Decode a BlockAgreement from r.
// Implements Msg interface.
func (b *BlockAgreement) Decode(r io.Reader) error {
	if err := encoding.Read256(r, &b.BlockHash); err != nil {
		return err
	}

	lVotes, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	b.VoteSet = make([]*Vote, lVotes)
	for i := uint64(0); i < lVotes; i++ {
		b.VoteSet[i] = &Vote{}
		if err := b.VoteSet[i].Decode(r); err != nil {
			return err
		}
	}

	return nil
}

// Type returns the consensus payload identifier.
// Implements Msg interface.
func (b *BlockAgreement) Type() ID {
	return BlockAgreementID
}
