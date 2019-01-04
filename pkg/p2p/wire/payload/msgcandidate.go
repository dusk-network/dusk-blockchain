package payload

import (
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// MsgCandidate defines a candidate message on the Dusk wire protocol.
type MsgCandidate struct {
	CandidateHash []byte         // Hash of the candidate block (32 bytes)
	SigBLS        *bls.Sig       // BLS signature (32 bytes)
	PubKey        *bls.PublicKey // Sender public key (32 bytes)
}

// NewMsgCandidate returns a MsgCandidate struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewMsgCandidate(candidateHash []byte, sig *bls.Sig, pubKey *bls.PublicKey) (*MsgCandidate, error) {
	if len(candidateHash) != 32 {
		return nil, errors.New("wire: supplied candidate hash for candidate message is improper length")
	}

	return &MsgCandidate{
		CandidateHash: candidateHash,
		SigBLS:        sig,
		PubKey:        pubKey,
	}, nil
}

// Encode a MsgCandidate struct and write to w.
// Implements Payload interface.
func (m *MsgCandidate) Encode(w io.Writer) error {
	if err := encoding.Write256(w, m.CandidateHash); err != nil {
		return err
	}

	// if err := encoding.Write512(w, m.SigBLS); err != nil {
	// 	return err
	// }

	pubBLS, err := m.PubKey.MarshalBinary()
	if err != nil {
		return err
	}

	if err := encoding.Write256(w, pubBLS); err != nil {
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

	// if err := encoding.Read512(r, &m.SigBLS); err != nil {
	// 	return err
	// }

	var pub []byte
	if err := encoding.Read256(r, &pub); err != nil {
		return err
	}

	m.PubKey.UnmarshalBinary(pub)

	return nil
}

// Command returns the command string associated with the candidate message.
// Implements payload interface.
func (m *MsgCandidate) Command() commands.Cmd {
	return commands.Candidate
}
