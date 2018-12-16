package payload

import (
	"fmt"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/wire/commands"
)

// MsgGetData defines a getdata message on the Dusk wire protocol.
type MsgGetData struct {
	Vectors []InvVect
}

// NewMsgGetData will return an empty MsgGetData struct.
// This can then be populated with the data to be requested
// via AddTx and AddBlock.
func NewMsgGetData() *MsgGetData {
	return &MsgGetData{}
}

// AddTx will add a tx inventory vector to MsgGetData.
func (m *MsgGetData) AddTx(tx transactions.Stealth) {
	vect := InvVect{
		Type: InvTx,
		Hash: tx.R,
	}

	m.Vectors = append(m.Vectors, vect)
}

// AddBlock will add a block inventory vector to MsgGetData.
func (m *MsgGetData) AddBlock(hash []byte) {
	// Finish this when block structure is defined.
	vect := InvVect{
		Type: InvBlock,
		Hash: hash,
	}

	m.Vectors = append(m.Vectors, vect)
}

// Encode a MsgGetData struct and write to w.
// Implements payload interface.
func (m *MsgGetData) Encode(w io.Writer) error {
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

// Decode a MsgGetData from r.
// Implements payload interface.
func (m *MsgGetData) Decode(r io.Reader) error {
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
			return fmt.Errorf("invalid inventory vector type %v", t)
		}

		m.Vectors[i].Type = InvType(t)
		if err := encoding.Read256(r, &m.Vectors[i].Hash); err != nil {
			return err
		}
	}

	return nil
}

// Command returns the command string associated with the getdata message.
// Implements payload interface.
func (m *MsgGetData) Command() commands.Cmd {
	return commands.GetData
}
