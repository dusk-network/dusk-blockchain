package consensusmsg

import (
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// SigSetVote defines a sigsetvote message on the Dusk wire protocol.
type SigSetVote struct {
	Step         uint8  // Current step
	SignatureSet []byte // Signature set voted on
	SigBLS       []byte // BLS signature of the signature set
	PubKeyBLS    []byte // Sender BLS public key
}

// NewSigSetVote returns a SigSetVote struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewSigSetVote(step uint8, sigSet, sigBLS, pubKeyBLS []byte) (*SigSetVote, error) {
	if len(sigBLS) != 32 {
		return nil, errors.New("wire: supplied BLS signature for signature set vote payload is improper length")
	}

	if len(pubKeyBLS) != 32 {
		return nil, errors.New("wire: supplied BLS public key for signature set vote payload is improper length")
	}

	return &SigSetVote{
		Step:         step,
		SignatureSet: sigSet,
		SigBLS:       sigBLS,
		PubKeyBLS:    pubKeyBLS,
	}, nil
}

// Encode a SigSetVote struct and write to w.
// Implements Msg interface.
func (s *SigSetVote) Encode(w io.Writer) error {
	if err := encoding.WriteUint8(w, s.Step); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, s.SignatureSet); err != nil {
		return err
	}

	if err := encoding.Write256(w, s.SigBLS); err != nil {
		return err
	}

	if err := encoding.Write256(w, s.PubKeyBLS); err != nil {
		return err
	}

	return nil
}

// Decode a SigSetVote from r.
// Implements Msg interface.
func (s *SigSetVote) Decode(r io.Reader) error {
	if err := encoding.ReadUint8(r, &s.Step); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.SignatureSet); err != nil {
		return err
	}

	if err := encoding.Read256(r, &s.SigBLS); err != nil {
		return err
	}

	if err := encoding.Read256(r, &s.PubKeyBLS); err != nil {
		return err
	}

	return nil
}

// Type returns the consensus payload identifier.
// Implements Msg interface.
func (s *SigSetVote) Type() ID {
	return SigSetVoteID
}
