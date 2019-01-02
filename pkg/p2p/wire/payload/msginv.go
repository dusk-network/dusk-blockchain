package payload

import (
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

// MsgInv defines an inv message on the Dusk wire protocol.
// It is used to broadcast new transactions and blocks.
type MsgInv struct {
	Vectors []*InvVect
}

// NewMsgInv will return an empty MsgInv struct. This struct can then
// be populated with the AddTx and AddBlock methods.
func NewMsgInv() *MsgInv {
	return &MsgInv{}
}

// AddTx will add an InvVect with type InvTx and the tx hash to
// MsgInv.Vectors.
func (m *MsgInv) AddTx(tx *transactions.Stealth) {
	vect := &InvVect{
		Type: InvTx,
		Hash: tx.Hash,
	}

	m.Vectors = append(m.Vectors, vect)
}

// AddBlock will add an InvVect with type InvBlock and the block
// hash to MsgInv.Vectors.
func (m *MsgInv) AddBlock(block *Block) {
	vect := &InvVect{
		Type: InvBlock,
		Hash: block.Header.Hash,
	}

	m.Vectors = append(m.Vectors, vect)
}

// Encode a MsgInv struct and write to w.
// Implements payload interface.
func (m *MsgInv) Encode(w io.Writer) error {
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

// Decode a MsgInv from r.
// Implements payload interface.
func (m *MsgInv) Decode(r io.Reader) error {
	n, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	m.Vectors = make([]*InvVect, n)
	for i := uint64(0); i < n; i++ {
		m.Vectors[i] = &InvVect{}
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

// Command returns the command string associated with the inv message.
// Implements payload interface.
func (m *MsgInv) Command() commands.Cmd {
	return commands.Inv
}
