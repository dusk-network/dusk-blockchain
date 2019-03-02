package consensusmsg

import (
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// CandidateScore defines a score message on the Dusk wire protocol.
type CandidateScore struct {
	Score         []byte
	Proof         []byte
	Z             []byte // Identity hash
	CandidateHash []byte // Block candidate hash (32 bytes)
	Seed          []byte // Seed of the current round
	PubList       []byte // Publist generated in the proof
}

// NewCandidateScore returns a CandidateScore struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewCandidateScore(score, proof, z, candidateHash, seed, pl []byte) (*CandidateScore, error) {
	if len(score) != 32 {
		return nil, errors.New("wire: supplied score for candidate score payload is improper length")
	}

	if len(z) != 32 {
		return nil, errors.New("wire: supplied z for candidate score payload is improper length")
	}

	if len(candidateHash) != 32 {
		return nil, errors.New("wire: supplied candidate hash for candidate score payload is improper length")
	}

	if len(seed) != 33 {
		return nil, errors.New("wire: supplied seed for candidate score payload is improper length")
	}

	return &CandidateScore{
		Score:         score,
		Proof:         proof,
		Z:             z,
		CandidateHash: candidateHash,
		Seed:          seed,
		PubList:       pl,
	}, nil
}

// Encode a CandidateScore struct and write to w.
// Implements Msg interface.
func (c *CandidateScore) Encode(w io.Writer) error {
	if err := encoding.Write256(w, c.Score); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, c.Proof); err != nil {
		return err
	}

	if err := encoding.Write256(w, c.Z); err != nil {
		return err
	}

	if err := encoding.Write256(w, c.CandidateHash); err != nil {
		return err
	}

	if err := encoding.WriteBLS(w, c.Seed); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, c.PubList); err != nil {
		return err
	}

	return nil
}

// Decode a CandidateScore from r.
// Implements Msg interface.
func (c *CandidateScore) Decode(r io.Reader) error {
	if err := encoding.Read256(r, &c.Score); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &c.Proof); err != nil {
		return err
	}

	if err := encoding.Read256(r, &c.Z); err != nil {
		return err
	}

	if err := encoding.Read256(r, &c.CandidateHash); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &c.Seed); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &c.PubList); err != nil {
		return err
	}

	return nil
}

// Type returns the consensus payload identifier.
// Implements Msg interface.
func (c *CandidateScore) Type() ID {
	return CandidateScoreID
}
