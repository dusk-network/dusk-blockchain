package payload

import (
	"encoding/binary"
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// MsgSigSetVote defines a sigsetvote message on the Dusk wire protocol.
type MsgSigSetVote struct {
	Stake         uint64 // Voter stake
	Round         uint64 // Current round
	Step          uint8  // Current step
	PrevBlockHash []byte // Previous block hash
	SignatureSet  []byte // Signature set voted on
	SigBLS        []byte // BLS signature of the signature set
	SigEd         []byte // Ed25519 signature of all above fields
	PubKeyBLS     []byte // Sender BLS public key
	PubKeyEd      []byte // Sender Ed25519 public key
}

// NewMsgSigSetVote returns a MsgSigSetVote struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewMsgSigSetVote(stake, round uint64, step uint8, prevBlock, sigSet, sigBLS, sigEd,
	pkBLS, pkEd []byte) (*MsgSigSetVote, error) {
	if len(prevBlock) != 32 {
		return nil, errors.New("wire: supplied previous block hash for signature set vote message is improper length")
	}

	if len(sigBLS) != 32 {
		return nil, errors.New("wire: supplied BLS signature for signature set vote message is improper length")
	}

	if len(sigEd) != 64 {
		return nil, errors.New("wire: supplied ed25519 signature for signature set vote message is improper length")
	}

	if len(pkBLS) != 32 {
		return nil, errors.New("wire: supplied BLS public key for signature set vote message is improper length")
	}

	if len(pkEd) != 32 {
		return nil, errors.New("wire: supplied ed25519 public key for signature set vote message is improper length")
	}

	return &MsgSigSetVote{
		Stake:         stake,
		Round:         round,
		Step:          step,
		PrevBlockHash: prevBlock,
		SignatureSet:  sigSet,
		SigEd:         sigEd,
		SigBLS:        sigBLS,
		PubKeyEd:      pkEd,
		PubKeyBLS:     pkBLS,
	}, nil
}

// Encode a MsgSigSetVote struct and write to w.
// Implements Payload interface.
func (m *MsgSigSetVote) Encode(w io.Writer) error {
	if err := encoding.WriteUint64(w, binary.LittleEndian, m.Stake); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, m.Round); err != nil {
		return err
	}

	if err := encoding.WriteUint8(w, m.Step); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.PrevBlockHash); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, m.SignatureSet); err != nil {
		return err
	}

	if err := encoding.Write512(w, m.SigEd); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.SigBLS); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.PubKeyEd); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.PubKeyBLS); err != nil {
		return err
	}

	return nil
}

// Decode a MsgSigSetVote from r.
// Implements Payload interface.
func (m *MsgSigSetVote) Decode(r io.Reader) error {
	if err := encoding.ReadUint64(r, binary.LittleEndian, &m.Stake); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &m.Round); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &m.Step); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.PrevBlockHash); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &m.SignatureSet); err != nil {
		return err
	}

	if err := encoding.Read512(r, &m.SigEd); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.SigBLS); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.PubKeyEd); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.PubKeyBLS); err != nil {
		return err
	}

	return nil
}

// Command returns the command string associated with the signature set vote message.
// Implements payload interface.
func (m *MsgSigSetVote) Command() commands.Cmd {
	return commands.SigSetVote
}
