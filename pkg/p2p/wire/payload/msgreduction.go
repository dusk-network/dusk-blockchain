package payload

import (
	"encoding/binary"
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// MsgReduction defines a reduction message on the Dusk wire protocol.
type MsgReduction struct {
	Score         *bls.Sig       // Sortition score of the sender
	Stake         uint64         // Sender's stake amount
	Round         uint64         // Current round
	Step          uint8          // Current step
	BlockHash     []byte         // Hash of the block being voted on (32 bytes)
	PrevBlockHash []byte         // Hash of the previous block (32 bytes)
	SigEd         []byte         // Ed25519 signature of the block hash, score, step, round and last block hash (64 bytes)
	SigBLS        *bls.Sig       // BLS signature of the voted block hash
	PubKeyEd      []byte         // Sender Ed25519 public key (32 bytes)
	PubKeyBLS     *bls.PublicKey // Sender BLS public key (32 bytes)
}

// NewMsgReduction returns a MsgReduction struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewMsgReduction(score *bls.Sig, hash, prevBlockHash, sigEd, pubKeyEd []byte, sigBLS *bls.Sig, pubKeyBLS *bls.PublicKey, stake, round uint64, step uint8) (*MsgReduction, error) {
	if len(hash) != 32 {
		return nil, errors.New("wire: supplied candidate hash for reduction message is improper length")
	}

	if len(pubKeyEd) != 32 {
		return nil, errors.New("wire: supplied pubkey for reduction message is improper length")
	}

	return &MsgReduction{
		Score:         score,
		Stake:         stake,
		Round:         round,
		Step:          step,
		BlockHash:     hash,
		PrevBlockHash: prevBlockHash,
		SigEd:         sigEd,
		SigBLS:        sigBLS,
		PubKeyEd:      pubKeyEd,
		PubKeyBLS:     pubKeyBLS,
	}, nil
}

// Encode a MsgReduction struct and write to w.
// Implements Payload interface.
func (m *MsgReduction) Encode(w io.Writer) error {
	// if err := encoding.Write256(w, m.Score); err != nil {
	// 	return err
	// }

	if err := encoding.WriteUint64(w, binary.LittleEndian, m.Stake); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, m.Round); err != nil {
		return err
	}

	if err := encoding.WriteUint8(w, m.Step); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.BlockHash); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.PrevBlockHash); err != nil {
		return err
	}

	if err := encoding.Write512(w, m.SigEd); err != nil {
		return err
	}

	// SigBLS

	if err := encoding.Write256(w, m.PubKeyEd); err != nil {
		return err
	}

	// PubKeyBLS

	return nil
}

// Decode a MsgReduction from r.
// Implements Payload interface.
func (m *MsgReduction) Decode(r io.Reader) error {
	// if err := encoding.Read256(r, &m.Score); err != nil {
	// 	return err
	// }

	if err := encoding.ReadUint64(r, binary.LittleEndian, &m.Stake); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &m.Round); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &m.Step); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.BlockHash); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.PrevBlockHash); err != nil {
		return err
	}

	if err := encoding.Read512(r, &m.SigEd); err != nil {
		return err
	}

	// SigBLS

	if err := encoding.Read256(r, &m.PubKeyEd); err != nil {
		return err
	}

	// PubKeyBLS

	return nil
}

// Command returns the command string associated with the reduction message.
// Implements payload interface.
func (m *MsgReduction) Command() commands.Cmd {
	return commands.Reduction
}
