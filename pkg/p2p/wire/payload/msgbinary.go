package payload

import (
	"encoding/binary"
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"golang.org/x/crypto/ed25519"
)

// MsgBinary defines a binary agreement message on the Dusk wire protocol.
type MsgBinary struct {
	Score         *bls.Sig           // Sortition score of sender
	Empty         bool               // Whether or not this block is an empty block
	Stake         uint64             // Sender's stake amount
	Round         uint64             // Current round
	Step          uint8              // Current step
	BlockHash     []byte             // Hash of the block being voted on (32 bytes)
	PrevBlockHash []byte             // Hash of the previous block (32 bytes)
	SigEd         []byte             // Ed25519 signature of the block hash, score, step, round and last block hash (64 bytes)
	SigBLS        *bls.Sig           // BLS signature of the voted block hash
	PubKeyEd      *ed25519.PublicKey // Sender Ed25519 public key (32 bytes)
	PubKeyBLS     *bls.PublicKey     // Sender BLS public key (32 bytes)
}

// NewMsgBinary returns a MsgBinary struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewMsgBinary(score *bls.Sig, empty bool, hash, prevBlockHash, sigEd []byte, pubKeyEd *ed25519.PublicKey,
	sigBLS *bls.Sig, pubKeyBLS *bls.PublicKey, stake, round uint64, step uint8) (*MsgBinary, error) {
	if len(hash) != 32 {
		return nil, errors.New("wire: supplied candidate hash for binary message is improper length")
	}

	if len(sigEd) != 64 {
		return nil, errors.New("wire: supplied sig for binary message is improper length")
	}

	return &MsgBinary{
		Score:         score,
		Empty:         empty,
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

// Encode a MsgBinary struct and write to w.
// Implements Payload interface.
func (m *MsgBinary) Encode(w io.Writer) error {
	// if err := encoding.Write256(w, m.Score); err != nil {
	// 	return err
	// }

	if err := encoding.WriteBool(w, m.Empty); err != nil {
		return err
	}

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

	if err := encoding.Write256(w, []byte(*m.PubKeyEd)); err != nil {
		return err
	}

	pubBLS, err := m.PubKeyBLS.MarshalBinary()
	if err != nil {
		return err
	}

	if err := encoding.Write256(w, pubBLS); err != nil {
		return err
	}

	return nil
}

// Decode a MsgBinary from r.
// Implements Payload interface.
func (m *MsgBinary) Decode(r io.Reader) error {
	// if err := encoding.Read256(r, &m.Score); err != nil {
	// 	return err
	// }

	if err := encoding.ReadBool(r, &m.Empty); err != nil {
		return err
	}

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

	var pubEd []byte
	if err := encoding.Read256(r, &pubEd); err != nil {
		return err
	}

	*m.PubKeyEd = ed25519.PublicKey(pubEd)

	var pubBLS []byte
	if err := encoding.Read256(r, &pubBLS); err != nil {
		return err
	}

	if err := m.PubKeyBLS.UnmarshalBinary(pubBLS); err != nil {
		return err
	}

	return nil
}

// Command returns the command string associated with the binary message.
// Implements payload interface.
func (m *MsgBinary) Command() commands.Cmd {
	return commands.Binary
}
