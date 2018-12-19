package payload

import (
	"encoding/binary"
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// MsgCertificateReq defines a certificatereq message on the Dusk wire protocol.
type MsgCertificateReq struct {
	BlockHeight uint64 // Block height of the requested certificate
	BlockHash   []byte // Block hash of the requested certificate (32 bytes)
}

// NewMsgCertificateReq returns a MsgCertificateReq struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewMsgCertificateReq(height uint64, hash []byte) (*MsgCertificateReq, error) {
	if len(hash) != 32 {
		return nil, errors.New("wire: supplied block hash for certificatereq message is improper length")
	}

	return &MsgCertificateReq{
		BlockHeight: height,
		BlockHash:   hash,
	}, nil
}

// Encode a MsgCertificateReq struct and write to w.
// Implements Payload interface.
func (m *MsgCertificateReq) Encode(w io.Writer) error {
	if err := encoding.WriteUint64(w, binary.LittleEndian, m.BlockHeight); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.BlockHash); err != nil {
		return err
	}

	return nil
}

// Decode a MsgCertificateReq from r.
// Implements Payload interface.
func (m *MsgCertificateReq) Decode(r io.Reader) error {
	if err := encoding.ReadUint64(r, binary.LittleEndian, &m.BlockHeight); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.BlockHash); err != nil {
		return err
	}

	return nil
}

// Command returns the command string associated with the certificatereq message.
// Implements payload interface.
func (m *MsgCertificateReq) Command() commands.Cmd {
	return commands.CertificateReq
}
