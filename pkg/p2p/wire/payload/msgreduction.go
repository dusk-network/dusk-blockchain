package payload

import (
	"encoding/binary"
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// MsgReduction defines a reduction message on the Dusk wire protocol.
type MsgReduction struct {
	Score     uint64
	BlockHash []byte // Hash of the block being reduced (32 bytes)
	SigEd     []byte // Ed25519 signature of the blok hash (64 bytes)
	PubKey    []byte // Sender public key (32 bytes)
}

// NewMsgReduction returns a MsgReduction struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewMsgReduction(score uint64, hash, sig, pubKey []byte) (*MsgReduction, error) {
	if len(hash) != 32 {
		return nil, errors.New("wire: supplied candidate hash for reduction message is improper length")
	}

	if len(sig) != 64 {
		return nil, errors.New("wire: supplied sig for reduction message is improper length")
	}

	if len(pubKey) != 32 {
		return nil, errors.New("wire: supplied pubkey for reduction message is improper length")
	}

	return &MsgReduction{
		Score:     score,
		BlockHash: hash,
		SigEd:     sig,
		PubKey:    pubKey,
	}, nil
}

// Encode a MsgReduction struct and write to w.
// Implements Payload interface.
func (m *MsgReduction) Encode(w io.Writer) error {
	if err := encoding.WriteUint64(w, binary.LittleEndian, m.Score); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.BlockHash); err != nil {
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

// Decode a MsgReduction from r.
// Implements Payload interface.
func (m *MsgReduction) Decode(r io.Reader) error {
	if err := encoding.ReadUint64(r, binary.LittleEndian, &m.Score); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.BlockHash); err != nil {
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

// Command returns the command string associated with the reduction message.
// Implements payload interface.
func (m *MsgReduction) Command() commands.Cmd {
	return commands.Reduction
}
