package payload

import (
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/wire/commands"
)

// MsgNotFound defines a notfound message on the Dusk wire protocol.
type MsgNotFound struct {
	Vectors []InvVect
}

// NewMsgNotFound returns an empty MsgNotFound struct.
func NewMsgNotFound() *MsgNotFound {
	return &MsgNotFound{}
}

// AddVector adds an inventory vector to MsgNotFound.
func (m *MsgNotFound) AddVector(vect InvVect) {
	m.Vectors = append(m.Vectors, vect)
}

// Encode a MsgNotFound struct and write to w.
// Implements payload interface.
func (m *MsgNotFound) Encode(w io.Writer) error {
	if err := encoding.WriteVarInt(w, uint64(len(m.Vectors))); err != nil {
		return err
	}

	for _, vect := range m.Vectors {
		if err := encoding.WriteUint8(w, uint8(vect.Type)); err != nil {
			return err
		}

		if err := encoding.Write256(w, vect.Hash); err != nil {
			return err
		}
	}

	return nil
}

// Decode a MsgNotFound from r.
// Implements payload interface.
func (m *MsgNotFound) Decode(r io.Reader) error {
	n, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	m.Vectors = make([]InvVect, n)
	for i := uint64(0); i < n; i++ {
		var t uint8
		if err := encoding.ReadUint8(r, &t); err != nil {
			return err
		}

		if InvType(t) != InvTx && InvType(t) != InvBlock {
			return errors.New("inventory vector type unknown")
		}

		m.Vectors[i].Type = InvType(t)
		if err := encoding.Read256(r, &m.Vectors[i].Hash); err != nil {
			return err
		}
	}

	return nil
}

// Command returns the command string associated with the notfound message.
// Implements payload interface.
func (m *MsgNotFound) Command() commands.Cmd {
	return commands.NotFound
}
