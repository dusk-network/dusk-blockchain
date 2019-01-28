package consensusmsg

import (
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// SigSetVote defines a sigsetvote message on the Dusk wire protocol.
type SigSetVote struct {
	Step             uint8  // Current step
	WinningBlockHash []byte // Hash of the winning block
	SigSetHash       []byte // Hash of signature set voted on
	SigBLS           []byte // Compressed BLS signature of the signature set
	PubKeyBLS        []byte // Sender BLS public key
	Score            []byte // Node sortition score
}

// NewSigSetVote returns a SigSetVote struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewSigSetVote(step uint8, winningBlock, sigSetHash, sigBLS, pubKeyBLS, score []byte) (*SigSetVote, error) {
	if len(winningBlock) != 32 {
		return nil, errors.New("wire: supplied winning block hash for signature set vote payload is improper length")
	}

	if len(sigSetHash) != 32 {
		return nil, errors.New("wire: supplied signature set hash for signature set vote payload is improper length")
	}

	if len(sigBLS) != 33 {
		return nil, errors.New("wire: supplied compressed BLS signature for signature set vote payload is improper length")
	}

	if len(score) != 33 {
		return nil, errors.New("wire: supplied score for signature set vote payload is improper length")
	}

	return &SigSetVote{
		Step:             step,
		WinningBlockHash: winningBlock,
		SigSetHash:       sigSetHash,
		SigBLS:           sigBLS,
		PubKeyBLS:        pubKeyBLS,
		Score:            score,
	}, nil
}

// Encode a SigSetVote struct and write to w.
// Implements Msg interface.
func (s *SigSetVote) Encode(w io.Writer) error {
	if err := encoding.WriteUint8(w, s.Step); err != nil {
		return err
	}

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

	if err := encoding.WriteBLS(w, s.Score); err != nil {
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

	if err := encoding.ReadBLS(r, &s.Score); err != nil {
		return err
	}

	return nil
}

// Type returns the consensus payload identifier.
// Implements Msg interface.
func (s *SigSetVote) Type() ID {
	return SigSetVoteID
}
