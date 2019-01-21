package payload

import (
	"encoding/binary"
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// MsgSignatureSet defines a signatureset message on the Dusk wire protocol.
type MsgSignatureSet struct {
	Stake            uint64 // Provisioner stake
	Round            uint64 // Current round
	PrevBlockHash    []byte // Previous block hash
	WinningBlockHash []byte // Winning block hash
	SignatureSet     []byte // Generated signature set
	Signature        []byte // Signature of all fields above
	PubKey           []byte // Ed25519 public key to verify the above signature
}

// NewMsgSignatureSet returns a MsgSignatureSet struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewMsgSignatureSet(stake, round uint64, prevBlock, winningBlock, sigSet, sig, pk []byte) (*MsgSignatureSet, error) {
	if len(prevBlock) != 32 {
		return nil, errors.New("wire: supplied previous block hash for signature set message is improper length")
	}

	if len(winningBlock) != 32 {
		return nil, errors.New("wire: supplied winning block hash for signature set message is improper length")
	}

	if len(sig) != 64 {
		return nil, errors.New("wire: supplied ed25519 signature for signature set message is improper length")
	}

	if len(pk) != 32 {
		return nil, errors.New("wire: supplied ed25519 public key for signature set message is improper length")
	}

	return &MsgSignatureSet{
		Stake:            stake,
		Round:            round,
		PrevBlockHash:    prevBlock,
		WinningBlockHash: winningBlock,
		SignatureSet:     sigSet,
		Signature:        sig,
		PubKey:           pk,
	}, nil
}

// Encode a MsgSignatureSet struct and write to w.
// Implements Payload interface.
func (m *MsgSignatureSet) Encode(w io.Writer) error {
	if err := encoding.WriteUint64(w, binary.LittleEndian, m.Stake); err != nil {
		return err
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, m.Round); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.PrevBlockHash); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.WinningBlockHash); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, m.SignatureSet); err != nil {
		return err
	}

	if err := encoding.Write512(w, m.Signature); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.PubKey); err != nil {
		return err
	}

	return nil
}

// Decode a MsgSignatureSet from r.
// Implements Payload interface.
func (m *MsgSignatureSet) Decode(r io.Reader) error {
	if err := encoding.ReadUint64(r, binary.LittleEndian, &m.Stake); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &m.Round); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.PrevBlockHash); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.WinningBlockHash); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &m.SignatureSet); err != nil {
		return err
	}

	if err := encoding.Read512(r, &m.Signature); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.PubKey); err != nil {
		return err
	}

	return nil
}

// Command returns the command string associated with the signature set message.
// Implements payload interface.
func (m *MsgSignatureSet) Command() commands.Cmd {
	return commands.SignatureSet
}
