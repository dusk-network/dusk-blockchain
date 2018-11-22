package payload

import (
	"fmt"
	"io"

	"github.com/toghrulmaharramov/dusk-go/encoding"
	"github.com/toghrulmaharramov/dusk-go/transactions"
	"github.com/toghrulmaharramov/dusk-go/wire/commands"
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

// AddBlock (add when block structure is defined)

// Encode a MsgGetData struct and write to w.
// Implements payload interface.
func (m *MsgGetData) Encode(w io.Writer) error {
	if err := encoding.WriteVarInt(w, uint64(len(m.Vectors))); err != nil {
		return err
	}

	for _, vect := range m.Vectors {
		if err := encoding.PutUint8(w, uint8(vect.Type)); err != nil {
			return err
		}

		if err := encoding.WriteHash(w, vect.Hash); err != nil {
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
		t, err := encoding.Uint8(r)
		if err != nil {
			return err
		}

		if InvType(t) != InvTx && InvType(t) != InvBlock {
			return fmt.Errorf("invalid inventory vector type %v", t)
		}

		h, err := encoding.ReadHash(r)
		if err != nil {
			return err
		}

		m.Vectors[i] = InvVect{
			Type: InvType(t),
			Hash: h,
		}
	}

	return nil
}

// Command returns the command string associated with the getdata message.
// Implements payload interface.
func (m *MsgGetData) Command() commands.Cmd {
	return commands.GetData
}
