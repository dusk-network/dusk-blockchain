package payload

import (
	"errors"
	"io"

	"github.com/toghrulmaharramov/dusk-go/encoding"
	"github.com/toghrulmaharramov/dusk-go/wire/commands"
)

// MsgCandidate defines a candidate message on the Dusk wire protocol.
type MsgCandidate struct {
	CandidateHash []byte // Hash of the candidate block (32 bytes)
	SigEd         []byte // Ed25519 signature (64 bytes)
	PubKey        []byte // Sender public key (32 bytes)
}

// NewMsgCandidate returns a MsgCandidate struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewMsgCandidate(candidateHash, sig, pubKey []byte) (*MsgCandidate, error) {
	if len(candidateHash) != 32 {
		return nil, errors.New("wire: supplied candidate hash for candidate message is improper length")
	}

	if len(sig) != 64 {
		return nil, errors.New("wire: supplied sig for candidate message is improper length")
	}

	if len(pubKey) != 32 {
		return nil, errors.New("wire: supplied pubkey for candidate message is improper length")
	}

	return &MsgCandidate{
		CandidateHash: candidateHash,
		SigEd:         sig,
		PubKey:        pubKey,
	}, nil
}

// Encode a MsgCandidate struct and write to w.
// Implements Payload interface.
func (m *MsgCandidate) Encode(w io.Writer) error {
	if err := encoding.Write256(w, m.CandidateHash); err != nil {
		return err
	}

	if err := encoding.Write512(w, m.SigEd); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.PubKey); err != nil {
		return err
	}

	return nil
}

// Decode a MsgCandidate from r.
// Implements Payload interface.
func (m *MsgCandidate) Decode(r io.Reader) error {
	if err := encoding.Read256(r, &m.CandidateHash); err != nil {
		return err
	}

	if err := encoding.Read512(r, &m.SigEd); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.PubKey); err != nil {
		return err
	}

	return nil
}

// Command returns the command string associated with the candidate message.
// Implements payload interface.
func (m *MsgCandidate) Command() commands.Cmd {
	return commands.Candidate
}
