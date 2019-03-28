package messages

import (
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// InvMessage defines an inv message on the Dusk wire protocol.
// It is used to broadcast new transactions and blocks.
type InvMessage struct {
	Vectors []*InvVect
}

// AddTx will add an InvVect with type InvTx and the tx hash to
// InvMessage.Vectors.
func (m *InvMessage) AddTx(hash []byte) error {
	vect := &InvVect{
		Type: InvTx,
		Hash: hash,
	}

	m.Vectors = append(m.Vectors, vect)
	return nil
}

// AddBlock will add an InvVect with type InvBlock and the block
// hash to InvMessage.Vectors.
func (m *InvMessage) AddBlock(hash []byte) {
	vect := &InvVect{
		Type: InvBlock,
		Hash: hash,
	}

	m.Vectors = append(m.Vectors, vect)
}

// Encode a InvMessage struct and write to w.
// Implements payload interface.
func (m *InvMessage) Encode(w io.Writer) error {
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

// Decode a InvMessage from r.
// Implements payload interface.
func (m *InvMessage) Decode(r io.Reader) error {
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
