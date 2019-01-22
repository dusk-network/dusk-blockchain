package consensusmsg

import (
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Agreement defines a binary agreement consensus payload on the Dusk wire protocol.
type Agreement struct {
	Score     []byte // Sortition score of sender
	Empty     bool   // Whether or not this block is an empty block
	Step      uint8  // Current step
	BlockHash []byte // Hash of the block being voted on (32 bytes)
	SigBLS    []byte // BLS signature of the voted block hash
	PubKeyBLS []byte // Sender BLS public key (32 bytes)
}

// NewAgreement returns an Agreement struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewAgreement(score []byte, empty bool, step uint8, hash, sigBLS, pubKeyBLS []byte) (*Agreement, error) {
	if len(score) != 32 {
		return nil, errors.New("wire: supplied score for agreement payload is improper length")
	}

	if len(hash) != 32 {
		return nil, errors.New("wire: supplied candidate hash for agreement payload is improper length")
	}

	if len(sigBLS) != 32 {
		return nil, errors.New("wire: supplied BLS signature for agreement payload is improper length")
	}

	if len(pubKeyBLS) != 32 {
		return nil, errors.New("wire: supplied BLS public key for agreement payload is improper length")
	}

	return &Agreement{
		Score:     score,
		Empty:     empty,
		Step:      step,
		BlockHash: hash,
		SigBLS:    sigBLS,
		PubKeyBLS: pubKeyBLS,
	}, nil
}

// Encode an Agreement struct and write to w.
// Implements Msg interface.
func (a *Agreement) Encode(w io.Writer) error {
	if err := encoding.Write256(w, a.Score); err != nil {
		return err
	}

	if err := encoding.WriteBool(w, a.Empty); err != nil {
		return err
	}

	if err := encoding.WriteUint8(w, a.Step); err != nil {
		return err
	}

	if err := encoding.Write256(w, a.BlockHash); err != nil {
		return err
	}

	if err := encoding.Write256(w, a.SigBLS); err != nil {
		return err
	}

	if err := encoding.Write256(w, a.PubKeyBLS); err != nil {
		return err
	}

	return nil
}

// Decode an Agreement from r.
// Implements Msg interface.
func (a *Agreement) Decode(r io.Reader) error {

	if err := encoding.Read256(r, &a.Score); err != nil {
		return err
	}

	if err := encoding.ReadBool(r, &a.Empty); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &a.Step); err != nil {
		return err
	}

	if err := encoding.Read256(r, &a.BlockHash); err != nil {
		return err
	}

	if err := encoding.Read256(r, &a.SigBLS); err != nil {
		return err
	}

	if err := encoding.Read256(r, &a.PubKeyBLS); err != nil {
		return err
	}

	return nil
}

// Type returns the consensus payload identifier.
// Implements Msg interface.
func (a *Agreement) Type() ID {
	return AgreementID
}
