package payload

import (
	"encoding/binary"
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// MsgCertificate defines a certificate message on the Dusk wire protocol.
type MsgCertificate struct {
	BlockHeight uint64       // Block height of the requested certificate
	BlockHash   []byte       // Block hash of the requested certificate (32 bytes)
	BlockCert   *Certificate // Block certificate (variable size)
}

// NewMsgCertificate returns a MsgCertificate struct populated with the specified information.
// This function provides checks for fixed-size fields, and will return an error
// if the checks fail.
func NewMsgCertificate(height uint64, hash []byte, cert *Certificate) (*MsgCertificate, error) {
	if len(hash) != 32 {
		return nil, errors.New("wire: supplied block hash for certificate message is improper length")
	}

	return &MsgCertificate{
		BlockHeight: height,
		BlockHash:   hash,
		BlockCert:   cert,
	}, nil
}

// Encode a MsgCertificate struct and write to w.
// Implements Payload interface.
func (m *MsgCertificate) Encode(w io.Writer) error {
	if err := encoding.WriteUint64(w, binary.LittleEndian, m.BlockHeight); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.BlockHash); err != nil {
		return err
	}

	if err := m.BlockCert.Encode(w); err != nil {
		return err
	}

	return nil
}

// Decode a MsgCertificate from r.
// Implements Payload interface.
func (m *MsgCertificate) Decode(r io.Reader) error {
	if err := encoding.ReadUint64(r, binary.LittleEndian, &m.BlockHeight); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.BlockHash); err != nil {
		return err
	}

	m.BlockCert = &Certificate{}
	if err := m.BlockCert.Decode(r); err != nil {
		return err
	}

	return nil
}

// Command returns the command string associated with the certificate message.
// Implements payload interface.
func (m *MsgCertificate) Command() commands.Cmd {
	return commands.Certificate
}
