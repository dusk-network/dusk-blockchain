package peer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/util"
)

// VersionMessage is a version message on the dusk wire protocol.
type VersionMessage struct {
	Version     *protocol.Version
	Timestamp   int64
	FromAddress *wire.NetAddress
	ToAddress   *wire.NetAddress
	Services    protocol.ServiceFlag
}

// We are trying to connect to another peer.
// We will send our Version with a MsgVerAck.
func (p *Writer) HandShake() error {
	if err := p.writeLocalMsgVersion(); err != nil {
		return err
	}

	if err := p.readVerack(); err != nil {
		return err
	}

	if err := p.readRemoteMsgVersion(); err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	verAckMessage, err := addHeader(buf, p.magic, topics.VerAck)
	if err != nil {
		return err
	}

	if _, err := p.Conn.Write(verAckMessage.Bytes()); err != nil {
		return err
	}

	return nil
}

func (p *Reader) HandShake() error {
	if err := p.readRemoteMsgVersion(); err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	verAckMessage, err := addHeader(buf, p.magic, topics.VerAck)
	if err != nil {
		return err
	}

	// We skip the message queue here and write immediately on the connection,
	// as the write loop has not been started yet.
	if _, err := p.Conn.Write(verAckMessage.Bytes()); err != nil {
		return err
	}

	if err := p.writeLocalMsgVersion(); err != nil {
		return err
	}

	if err := p.readVerack(); err != nil {
		return err
	}
	return nil
}

func (p *Connection) writeLocalMsgVersion() error {
	fromPort := uint16(p.Port())
	version := protocol.NodeVer
	localIP, err := util.GetOutboundIP()
	if err != nil {
		return err
	}
	fromAddr := wire.NetAddress{
		IP:   localIP,
		Port: fromPort,
	}

	toIP := p.Conn.RemoteAddr().(*net.TCPAddr).IP
	toPort := p.Conn.RemoteAddr().(*net.TCPAddr).Port
	toAddr := wire.NewNetAddress(toIP.String(), uint16(toPort))

	message, err := newVersionMessageBuffer(version, &fromAddr, toAddr, protocol.FullNode)
	if err != nil {
		return err
	}

	messageWithHeader, err := addHeader(message, p.magic, topics.Version)
	if err != nil {
		return err
	}

	_, err = p.Conn.Write(messageWithHeader.Bytes())
	return err
}

func (p *Connection) readRemoteMsgVersion() error {
	header, err := p.readHeader()
	if err != nil {
		return err
	}

	if header.Topic != topics.Version {
		return fmt.Errorf("did not receive the expected '%s' message - got %s",
			topics.Version, header.Topic)
	}

	buffer := make([]byte, header.Length)
	if _, err := io.ReadFull(p.Conn, buffer); err != nil {
		return err
	}

	version, err := decodeVersionMessage(bytes.NewBuffer(buffer))

	if err != nil {
		return err
	}

	return verifyVersion(version.Version)
}

func (p *Connection) readVerack() error {
	header, err := p.readHeader()
	if err != nil {
		return err
	}

	if header.Topic != topics.VerAck {
		return fmt.Errorf("did not receive the expected '%s' message - got %s",
			topics.VerAck, header.Topic)
	}

	return nil
}

func newVersionMessageBuffer(v *protocol.Version, from, to *wire.NetAddress,
	services protocol.ServiceFlag) (*bytes.Buffer, error) {

	buffer := new(bytes.Buffer)
	if err := v.Encode(buffer); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint64(buffer, binary.LittleEndian,
		uint64(time.Now().Unix())); err != nil {
		return nil, err
	}

	if err := from.Encode(buffer); err != nil {
		return nil, err
	}

	if err := to.Encode(buffer); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint64(buffer, binary.LittleEndian, uint64(services)); err != nil {
		return nil, err
	}

	return buffer, nil
}
func decodeVersionMessage(r *bytes.Buffer) (*VersionMessage, error) {
	versionMessage := &VersionMessage{
		Version:     &protocol.Version{},
		FromAddress: &wire.NetAddress{},
		ToAddress:   &wire.NetAddress{},
	}
	if err := versionMessage.Version.Decode(r); err != nil {
		return nil, err
	}

	var time uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &time); err != nil {
		return nil, err
	}

	versionMessage.Timestamp = int64(time)

	if err := versionMessage.FromAddress.Decode(r); err != nil {
		return nil, err
	}

	if err := versionMessage.ToAddress.Decode(r); err != nil {
		return nil, err
	}

	var services uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &services); err != nil {
		return nil, err
	}

	versionMessage.Services = protocol.ServiceFlag(services)
	return versionMessage, nil
}

func verifyVersion(v *protocol.Version) error {
	if protocol.NodeVer.Major != v.Major {
		return errors.New("version mismatch")
	}

	return nil
}
