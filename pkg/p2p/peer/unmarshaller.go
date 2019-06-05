package peer

import (
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

type messageUnmarshaller struct {
	magic protocol.Magic
}

func (m *messageUnmarshaller) Unmarshal(b []byte, w io.Writer) error {
	payloadBuf := processing.Decode(b)
	magic := extractMagic(payloadBuf)

	if !m.magicIsValid(magic) {
		return errors.New("received message header magic is mismatched")
	}

	if _, err := payloadBuf.WriteTo(w); err != nil {
		return err
	}

	return nil
}

func (m *messageUnmarshaller) magicIsValid(magic protocol.Magic) bool {
	return m.magic == magic
}
