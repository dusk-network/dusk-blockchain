package peer

import (
	"encoding/binary"
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

type messageUnmarshaller struct {
	magic protocol.Magic
}

func (m *messageUnmarshaller) Unmarshal(b []byte, w io.Writer) error {
	payloadBuf := Decode(b)
	magic := m.extractMagic(payloadBuf)

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

func (m *messageUnmarshaller) extractMagic(r io.Reader) protocol.Magic {
	buffer := make([]byte, 4)
	if _, err := io.ReadFull(r, buffer); err != nil {
		panic(err)
	}

	magic := binary.LittleEndian.Uint32(buffer)
	return protocol.Magic(magic)
}
