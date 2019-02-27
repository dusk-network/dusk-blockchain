package consensusmsg

import (
	"encoding/binary"
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Vote defines a block set or signature set vote for the consensus.
type Vote struct {
	Hash   []byte // The block hash or signature set hash voted on
	PubKey []byte // BLS public key of the voter
	Sig    []byte // Compressed BLS signature of the hash
	Step   uint32 // Step this vote occurred at
}

// NewVote will return a Vote struct populated with the passed parameters.
func NewVote(hash, pubKey, sig []byte, step uint32) (*Vote, error) {
	if len(hash) != 32 {
		return nil, errors.New("supplied hash for vote is improper length")
	}

	if len(sig) != 33 {
		return nil, errors.New("supplied sig for vote is improper length")
	}

	return &Vote{
		Hash:   hash,
		PubKey: pubKey,
		Sig:    sig,
		Step:   step,
	}, nil
}

// Encode a Vote struct to w.
func (v *Vote) Encode(w io.Writer) error {
	if err := encoding.Write256(w, v.Hash); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, v.PubKey); err != nil {
		return err
	}

	if err := encoding.WriteBLS(w, v.Sig); err != nil {
		return err
	}

	if err := encoding.WriteUint32(w, binary.LittleEndian, v.Step); err != nil {
		return err
	}

	return nil
}

// Decode a Vote struct from r.
func (v *Vote) Decode(r io.Reader) error {
	if err := encoding.Read256(r, &v.Hash); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &v.PubKey); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &v.Sig); err != nil {
		return err
	}

	if err := encoding.ReadUint32(r, binary.LittleEndian, &v.Step); err != nil {
		return err
	}

	return nil
}
