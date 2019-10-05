package peer

import (
	"bytes"
	"errors"
	"io"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
)

type messageUnmarshaller struct {
	magic protocol.Magic
}

func (m *messageUnmarshaller) Unmarshal(b []byte, w io.Writer) error {

	payloadBuf := new(bytes.Buffer)
	payloadBuf.Write(b)

	magic, err := protocol.Extract(payloadBuf)
	if err != nil {
		return err
	}

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
