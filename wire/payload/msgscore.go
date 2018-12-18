package payload

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/toghrulmaharramov/dusk-go/wire/commands"

	"github.com/toghrulmaharramov/dusk-go/encoding"
)

// MsgScore defines a score message on the Dusk wire protocol.
type MsgScore struct {
	Score         uint64
	Proof         []byte // variable size
	CandidateHash []byte // Block candidate hash (32 bytes)
	SigEd         []byte // Ed25519 signature of the score, proof and candidate hash (64 bytes)
	PubKey        []byte // Sender public key (32 bytes)
}

// NewMsgScore returns a MsgScore struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewMsgScore(score uint64, proof, candidateHash, sig, pubKey []byte) (*MsgScore, error) {
	if len(candidateHash) != 32 {
		return nil, errors.New("wire: supplied candidate hash for score message is improper length")
	}

	if len(sig) != 64 {
		return nil, errors.New("wire: supplied sig for score message is improper length")
	}

	if len(pubKey) != 32 {
		return nil, errors.New("wire: supplied pubkey for score message is improper length")
	}

	return &MsgScore{
		Score:         score,
		Proof:         proof,
		CandidateHash: candidateHash,
		SigEd:         sig,
		PubKey:        pubKey,
	}, nil
}

// Encode a MsgScore struct and write to w.
// Implements Payload interface.
func (m *MsgScore) Encode(w io.Writer) error {
	if err := encoding.WriteUint64(w, binary.LittleEndian, m.Score); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, m.Proof); err != nil {
		return err
	}

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

// Decode a MsgScore from r.
// Implements Payload interface.
func (m *MsgScore) Decode(r io.Reader) error {
	if err := encoding.ReadUint64(r, binary.LittleEndian, &m.Score); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &m.Proof); err != nil {
		return err
	}

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

// Command returns the command string associated with the score message.
// Implements payload interface.
func (m *MsgScore) Command() commands.Cmd {
	return commands.Score
}
