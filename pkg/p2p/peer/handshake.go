package peer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
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

// Handshake either sends or receives a handshake.
// Sending involves writing a 'version' msg to an other peer.
// Receiving processes a received 'version' msg by sending our 'version' with a 'verack' msg.
func (p *Peer) Handshake() error {
	handshakeErr := make(chan error, 1)
	go func() {
		if p.Inbound {
			// An other peer wants to handshake with us.
			handshakeErr <- p.inboundHandShake()
		} else {
			// We want to handshake with an other peer.
			handshakeErr <- p.outboundHandShake()
		}
	}()

	select {
	case err := <-handshakeErr:
		if err != nil {
			return fmt.Errorf(errHandShakeFromStr, err)
		}
	case <-time.After(handshakeTimeout):
		return errHandShakeTimeout
	}

	return nil
}

// We are trying to connect to another peer.
// We will send our Version with a MsgVerAck.
func (p *Peer) outboundHandShake() error {
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

func (p *Peer) inboundHandShake() error {
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

	if err := p.writeLocalMsgVersion(); err != nil {
		return err
	}

	if err := p.readVerack(); err != nil {
		return err
	}
	return nil
}

func (p *Peer) writeLocalMsgVersion() error {
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

func (p *Peer) readRemoteMsgVersion() error {
	topic, payload, err := p.readMessage()
	if err != nil {
		return err
	}

	if topic != topics.Version {
		return fmt.Errorf("did not receive the expected '%s' message - got %s", topics.Version, topic)
	}

	version, err := decodeVersionMessage(payload)
	if err != nil {
		return err
	}

	return verifyVersion(version.Version)
}

func (p *Peer) readVerack() error {
	topic, _, err := p.readMessage()
	if err != nil {
		return err
	}

	if topic != topics.VerAck {
		return fmt.Errorf("did not receive the expected '%s' message - got %s", topics.VerAck, topic)
	}

	// should only be accessed on one go-routine
	p.VerackReceived = true

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
