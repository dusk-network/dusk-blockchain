package messages

import (
	"fmt"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// GetDataMessage defines a getdata message on the Dusk wire protocol.
type GetDataMessage struct {
	Vectors []*InvVect
}

// AddTx will add a tx inventory vector to GetDataMessage.
func (m *GetDataMessage) AddTx(hash []byte) {
	vect := &InvVect{
		Type: InvTx,
		Hash: hash,
	}

	m.Vectors = append(m.Vectors, vect)
}

// AddBlock will add a block inventory vector to GetDataMessage.
func (m *GetDataMessage) AddBlock(hash []byte) {
	vect := &InvVect{
		Type: InvBlock,
		Hash: hash,
	}

	m.Vectors = append(m.Vectors, vect)
}

// AddBlocks will add blocks inventory vector to GetDataMessage.
func (m *GetDataMessage) AddBlocks(blocks []*block.Block) {
	for _, block := range blocks {
		m.AddBlock(block.Header.Hash)
	}
}

// Encode a GetDataMessage struct and write to w.
// Implements payload interface.
func (m *GetDataMessage) Encode(w io.Writer) error {
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

// Decode a GetDataMessage from r.
// Implements payload interface.
func (m *GetDataMessage) Decode(r io.Reader) error {
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
			return fmt.Errorf("Invalid inventory vector type %v", t)
		}

		m.Vectors[i].Type = InvType(t)
		if err := encoding.Read256(r, &m.Vectors[i].Hash); err != nil {
			return err
		}
	}

	return nil
}
