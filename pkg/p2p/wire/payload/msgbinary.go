package payload

import (
	"encoding/binary"
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/wire/commands"
)

// MsgBinary defines a binary message on the Dusk wire protocol.
type MsgBinary struct {
	Score  uint64
	Agreed bool   // Whether or not this block is agreed upon by the sender
	Hash   []byte // The combined hash of the block hash + current round + current step (32 bytes)
	SigBLS []byte // Compressed BLS signature of the combined hash (32 bytes)
	SigEd  []byte // Ed25519 signature of the score, agreement, and BLS signature (64 bytes)
	PubKey []byte // Sender public key (32 bytes)
}

// NewMsgBinary returns a MsgBinary struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewMsgBinary(score uint64, agree bool, hash, sigBLS, sigEd, pubKey []byte) (*MsgBinary, error) {
	if len(hash) != 32 {
		return nil, errors.New("wire: supplied candidate hash for binary message is improper length")
	}

	if len(sigBLS) != 32 {
		return nil, errors.New("wire: supplied sig for binary message is improper length")
	}

	if len(sigEd) != 64 {
		return nil, errors.New("wire: supplied sig for binary message is improper length")
	}

	if len(pubKey) != 32 {
		return nil, errors.New("wire: supplied pubkey for binary message is improper length")
	}

	return &MsgBinary{
		Score:  score,
		Agreed: agree,
		Hash:   hash,
		SigBLS: sigBLS,
		SigEd:  sigEd,
		PubKey: pubKey,
	}, nil
}

// Encode a MsgBinary struct and write to w.
// Implements Payload interface.
func (m *MsgBinary) Encode(w io.Writer) error {
	if err := encoding.WriteUint64(w, binary.LittleEndian, m.Score); err != nil {
		return err
	}

	if err := encoding.WriteBool(w, m.Agreed); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.Hash); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.SigBLS); err != nil {
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

// Decode a MsgBinary from r.
// Implements Payload interface.
func (m *MsgBinary) Decode(r io.Reader) error {
	if err := encoding.ReadUint64(r, binary.LittleEndian, &m.Score); err != nil {
		return err
	}

	if err := encoding.ReadBool(r, &m.Agreed); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.Hash); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.SigBLS); err != nil {
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

// Command returns the command string associated with the binary message.
// Implements payload interface.
func (m *MsgBinary) Command() commands.Cmd {
	return commands.Binary
}
