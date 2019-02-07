package payload

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
)

// MsgHeaders defines a headers message on the Dusk wire protocol.
type MsgHeaders struct {
	Headers []*block.Header
}

// NewMsgHeaders will return an empty MsgHeaders struct.
func NewMsgHeaders() *MsgHeaders {
	return &MsgHeaders{}
}

// AddHeader will add a block header to the MsgHeaders struct.
func (m *MsgHeaders) AddHeader(header *block.Header) {
	m.Headers = append(m.Headers, header)
}

// Clear the MsgHeaders struct.
func (m *MsgHeaders) Clear() {
	m.Headers = nil
}

// Encode implements payload interface.
func (m *MsgHeaders) Encode(w io.Writer) error {
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

// Decode implements payload interface.
func (m *MsgHeaders) Decode(r io.Reader) error {
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

// Command returns the command string associated with the getheaders message.
// Implements payload interface.
func (m *MsgHeaders) Command() commands.Cmd {
	return commands.Headers
}
