package messages

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type GetHeadersMessage struct {
	Locator  []byte
	HashStop []byte
}

func (m *GetHeadersMessage) Encode(w io.Writer) error {
	if err := encoding.Write256(w, m.Locator); err != nil {
		return err
	}

	if err := encoding.Write256(w, m.HashStop); err != nil {
		return err
	}

	return nil
}

func (m *GetHeadersMessage) Decode(r io.Reader) error {
	if err := encoding.Read256(r, &m.Locator); err != nil {
		return err
	}

	if err := encoding.Read256(r, &m.HashStop); err != nil {
		return err
	}

	return nil
}
