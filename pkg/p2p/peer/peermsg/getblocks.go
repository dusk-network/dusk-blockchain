package peermsg

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type GetBlocks struct {
	Locators [][]byte
	Target   []byte
}

func (g *GetBlocks) Encode(w io.Writer) error {
	if err := encoding.WriteVarInt(w, uint64(len(g.Locators))); err != nil {
		return err
	}

	for _, locator := range g.Locators {
		if err := encoding.Write256(w, locator); err != nil {
			return err
		}
	}

	if err := encoding.Write256(w, g.Target); err != nil {
		return err
	}

	return nil
}

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

	if err := encoding.Read256(r, &g.Target); err != nil {
		return err
	}

	return nil
}
