package wire

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

// Payload defines the message payload.
type Payload interface {
	Encode(w io.Writer) error
	Decode(r io.Reader) error
	Command() commands.Cmd
}

// WriteMessage will write a Dusk wire message to w.
func WriteMessage(w io.Writer, magic protocol.DuskNetwork, p Payload) error {
	if err := encoding.WriteUint32(w, binary.LittleEndian, uint32(magic)); err != nil {
		return err
	}

	byteCmd := commands.CmdToByteArray(p.Command())
	if err := binary.Write(w, binary.LittleEndian, byteCmd); err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if err := p.Encode(buf); err != nil {
		return err
	}

	payloadLength := uint32(buf.Len())
	checksum, err := crypto.Checksum(buf.Bytes())
	if err != nil {
		return err
	}

	if err := encoding.WriteUint32(w, binary.LittleEndian, payloadLength); err != nil {
		return err
	}

	if err := encoding.WriteUint32(w, binary.LittleEndian, checksum); err != nil {
		return err
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return err
	}

	return nil
}

// ReadMessage will read a Dusk wire message from r and return the associated payload.
func ReadMessage(r io.Reader, magic protocol.DuskNetwork) (Payload, error) {
	buf := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}

	hdrBuf := bytes.NewReader(buf)
	var hdr Header
	if err := hdr.Decode(hdrBuf); err != nil {
		return nil, err
	}

	if magic != hdr.Magic {
		return nil, errors.New("magic mismatch")
	}

	pBuf := make([]byte, 0, hdr.Length)
	payloadBuf := bytes.NewBuffer(pBuf)
	if _, err := io.Copy(payloadBuf, r); err != nil {
		return nil, err
	}

	if !crypto.CompareChecksum(payloadBuf.Bytes(), hdr.Checksum) {
		return nil, errors.New("checksum mismatch")
	}

	switch hdr.Command {
	case commands.Version:
		m := &payload.MsgVersion{}
		err := m.Decode(payloadBuf)
		return m, err
	case commands.VerAck:
		return payload.NewMsgVerAck(), nil
	case commands.Ping:
		return payload.NewMsgPing(), nil
	case commands.Pong:
		return payload.NewMsgPong(), nil
	case commands.Addr:
		m := payload.NewMsgAddr()
		err := m.Decode(payloadBuf)
		return m, err
	case commands.GetAddr:
		return payload.NewMsgGetAddr(), nil
	case commands.GetData:
		m := payload.NewMsgGetData()
		err := m.Decode(payloadBuf)
		return m, err
	case commands.GetBlocks:
		m := &payload.MsgGetBlocks{}
		err := m.Decode(payloadBuf)
		return m, err
	case commands.GetHeaders:
		m := &payload.MsgGetHeaders{}
		err := m.Decode(payloadBuf)
		return m, err
	case commands.Tx:
		m := &payload.MsgTx{}
		err := m.Decode(payloadBuf)
		return m, err
	// case commands.Block:
	// case commands.Headers:
	case commands.MemPool:
		return payload.NewMsgMemPool(), nil
	case commands.Inv:
		m := payload.NewMsgInv()
		err := m.Decode(payloadBuf)
		return m, err
	case commands.CertificateReq:
		m := &payload.MsgCertificateReq{}
		err := m.Decode(payloadBuf)
		return m, err
	case commands.Certificate:
		m := &payload.MsgCertificate{}
		err := m.Decode(payloadBuf)
		return m, err
	case commands.Score:
		m := &payload.MsgScore{}
		err := m.Decode(payloadBuf)
		return m, err
	case commands.Candidate:
		m := &payload.MsgCandidate{}
		err := m.Decode(payloadBuf)
		return m, err
	case commands.Reduction:
		m := &payload.MsgReduction{}
		err := m.Decode(payloadBuf)
		return m, err
	case commands.Binary:
		m := &payload.MsgBinary{}
		err := m.Decode(payloadBuf)
		return m, err
	case commands.NotFound:
		m := payload.NewMsgNotFound()
		err := m.Decode(payloadBuf)
		return m, err
	case commands.Reject:
		m := &payload.MsgReject{}
		err := m.Decode(payloadBuf)
		return m, err
	default:
		return nil, fmt.Errorf("unknown command %v", hdr.Command)
	}
}
