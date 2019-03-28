package messages

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type HeadersMessage struct {
	Headers []*block.Header
}

func (m *HeadersMessage) Encode(w io.Writer) error {
	lHeaders := uint64(len(m.Headers))
	if err := encoding.WriteVarInt(w, lHeaders); err != nil {
		return err
	}

	for _, hdr := range m.Headers {
		if err := hdr.Encode(w); err != nil {
			return err
		}
	}

	return nil
}

func (m *HeadersMessage) Decode(r io.Reader) error {
	lHeaders, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	m.Headers = make([]*block.Header, lHeaders)
	for i := uint64(0); i < lHeaders; i++ {
		m.Headers[i] = &block.Header{}
		if err := m.Headers[i].Decode(r); err != nil {
			return err
		}
	}

	return nil
}
