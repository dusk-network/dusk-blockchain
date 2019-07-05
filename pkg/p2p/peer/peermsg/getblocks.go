package peermsg

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// GetBlocks defines a getblocks message on the Dusk wire protocol. It is used to
// request blocks from another peer.
type GetBlocks struct {
	Locators [][]byte
}

// Encode a GetBlocks struct and write it to w.
func (g *GetBlocks) Encode(w io.Writer) error {
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

// Decode a GetBlocks struct from r into g.
func (g *GetBlocks) Decode(r io.Reader) error {
	lenLocators, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	g.Locators = make([][]byte, lenLocators)
	for i := uint64(0); i < lenLocators; i++ {
		if err := encoding.Read256(r, &g.Locators[i]); err != nil {
			return err
		}
	}

	return nil
}
