package consensusmsg

import (
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// SigSetReduction defines a sigsetreduction message on the Dusk wire protocol.
type SigSetReduction struct {
	WinningBlockHash []byte // Hash of the winning block
	SigSetHash       []byte // Hash of signature set voted on
	SigBLS           []byte // Compressed BLS signature of the signature set
	PubKeyBLS        []byte // Sender BLS public key
}

// NewSigSetReduction returns a SigSetReduction struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewSigSetReduction(winningBlock, sigSetHash, sigBLS, pubKeyBLS []byte) (*SigSetReduction, error) {
	if len(winningBlock) != 32 {
		return nil, errors.New("wire: supplied winning block hash for signature set vote payload is improper length")
	}

	if len(sigSetHash) != 32 {
		return nil, errors.New("wire: supplied signature set hash for signature set vote payload is improper length")
	}

	if len(sigBLS) != 33 {
		return nil, errors.New("wire: supplied compressed BLS signature for signature set vote payload is improper length")
	}

	return &SigSetReduction{
		WinningBlockHash: winningBlock,
		SigSetHash:       sigSetHash,
		SigBLS:           sigBLS,
		PubKeyBLS:        pubKeyBLS,
	}, nil
}

// Encode a SigSetReduction struct and write to w.
// Implements Msg interface.
func (s *SigSetReduction) Encode(w io.Writer) error {
	if err := encoding.Write256(w, s.WinningBlockHash); err != nil {
		return err
	}

	if err := encoding.Write256(w, s.SigSetHash); err != nil {
		return err
	}

	if err := encoding.WriteBLS(w, s.SigBLS); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, s.PubKeyBLS); err != nil {
		return err
	}

	return nil
}

// Decode a SigSetReduction from r.
// Implements Msg interface.
func (s *SigSetReduction) Decode(r io.Reader) error {
	if err := encoding.Read256(r, &s.WinningBlockHash); err != nil {
		return err
	}

	if err := encoding.Read256(r, &s.SigSetHash); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &s.SigBLS); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &s.PubKeyBLS); err != nil {
		return err
	}

	return nil
}

// Type returns the consensus payload identifier.
// Implements Msg interface.
func (s *SigSetReduction) Type() ID {
	return SigSetReductionID
}
