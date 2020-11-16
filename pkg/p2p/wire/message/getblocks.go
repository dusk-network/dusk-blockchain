package message

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
)

// GetBlocks defines a getblocks message on the Dusk wire protocol. It is used to
// request blocks from another peer.
type GetBlocks struct {
	Locators [][]byte
}

// Copy a GetBlocks message.
// Implements the payload.Safe interface.
func (g GetBlocks) Copy() payload.Safe {
	l := make([][]byte, len(g.Locators))
	copy(l, g.Locators)
	return &GetBlocks{l}
}

// Encode a GetBlocks struct and write it to w.
func (g *GetBlocks) Encode(w *bytes.Buffer) error {
	if err := encoding.WriteVarInt(w, uint64(len(g.Locators))); err != nil {
		return err
	}

	for _, locator := range g.Locators {
		if err := encoding.Write256(w, locator); err != nil {
			return err
		}
	}

	return nil
}

// UnmarshalGetBlocksMessage unmarshals a GetBlocks message into a
// SerializableMessage.
func UnmarshalGetBlocksMessage(r *bytes.Buffer, m SerializableMessage) error {
	g := &GetBlocks{}
	if err := g.Decode(r); err != nil {
		return err
	}

	m.SetPayload(*g)
	return nil
}

// Decode a GetBlocks struct from r into g.
func (g *GetBlocks) Decode(r *bytes.Buffer) error {
	lenLocators, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	// lenLocators should never exceed 500, as that is the maximum amount
	// of blocks a peer can request
	// TODO: remove hardcoding
	if lenLocators > 500 {
		return errors.New("too many locators in GetBlocks message")
	}

	g.Locators = make([][]byte, lenLocators)
	for i := uint64(0); i < lenLocators; i++ {
		g.Locators[i] = make([]byte, 32)
		if err = encoding.Read256(r, g.Locators[i]); err != nil {
			return err
		}
	}

	return nil
}
