package consensusmsg

import (
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Reduction defines a reduction message on the Dusk wire protocol.
type Reduction struct {
	Score     []byte // Sortition score of the sender
	Step      uint8  // Current step
	BlockHash []byte // Hash of the block being voted on (32 bytes)
	SigBLS    []byte // BLS signature of the voted block hash
	PubKeyBLS []byte // Sender BLS public key (32 bytes)
}

// NewReduction returns a Reduction struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewReduction(score []byte, step uint8, hash, sigBLS, pubKeyBLS []byte) (*Reduction, error) {
	if len(score) != 32 {
		return nil, errors.New("wire: supplied score for reduction payload is improper length")
	}

	if len(hash) != 32 {
		return nil, errors.New("wire: supplied candidate hash for reduction payload is improper length")
	}

	if len(sigBLS) != 32 {
		return nil, errors.New("wire: supplied BLS signature for reduction payload is improper length")
	}

	if len(pubKeyBLS) != 32 {
		return nil, errors.New("wire: supplied BLS public key for reduction payload is improper length")
	}

	return &Reduction{
		Score:     score,
		Step:      step,
		BlockHash: hash,
		SigBLS:    sigBLS,
		PubKeyBLS: pubKeyBLS,
	}, nil
}

// Encode a Reduction struct and write to w.
// Implements Msg interface.
func (rd *Reduction) Encode(w io.Writer) error {
	if err := encoding.Write256(w, rd.Score); err != nil {
		return err
	}

	if err := encoding.WriteUint8(w, rd.Step); err != nil {
		return err
	}

	if err := encoding.Write256(w, rd.BlockHash); err != nil {
		return err
	}

	if err := encoding.Write256(w, rd.SigBLS); err != nil {
		return err
	}

	if err := encoding.Write256(w, rd.PubKeyBLS); err != nil {
		return err
	}

	return nil
}

// Decode a Reduction from r.
// Implements Msg interface.
func (rd *Reduction) Decode(r io.Reader) error {
	if err := encoding.Read256(r, &rd.Score); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &rd.Step); err != nil {
		return err
	}

	if err := encoding.Read256(r, &rd.BlockHash); err != nil {
		return err
	}

	if err := encoding.Read256(r, &rd.SigBLS); err != nil {
		return err
	}

	if err := encoding.Read256(r, &rd.PubKeyBLS); err != nil {
		return err
	}

	return nil
}

// Type returns the consensus payload identifier.
// Implements Msg interface.
func (rd *Reduction) Type() ID {
	return ReductionID
}
