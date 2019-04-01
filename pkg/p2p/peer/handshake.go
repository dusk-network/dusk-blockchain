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

// An other Peer wants to handshake with us.
// We will send our Version with a MsgVerAck.
func (p *Peer) inboundHandShake() error {
	var err error
	if err = p.writeLocalMsgVersion(); err != nil {
		fmt.Println("error writing version message,", err)
		return err
	}

	topic, payload, err := p.readMessage()
	if err != nil {
		fmt.Println("error reading message,", err)
		return err
	}

	if topic != topics.Version {
		return fmt.Errorf("received %s message, when we expected %s message", topic, topics.Version)
	}

	version, err := decodeVersionMessage(payload)
	if err != nil {
		fmt.Println("error decoding version message,", err)
		return err
	}

	if err := p.verifyVersion(version); err != nil {
		fmt.Println("error verifying version message,", err)
		return err
	}

	if err := p.WriteMessage(nil, topics.VerAck); err != nil {
		fmt.Println("error writing verack message,", err)
		return err
	}

	return p.readVerack()
}

func (p *Peer) outboundHandShake() error {
	if err := p.readRemoteMsgVersion(); err != nil {
		fmt.Println("error writing version message,", err)
		return err
	}

	if err := p.writeLocalMsgVersion(); err != nil {
		fmt.Println("error reading message,", err)
		return err
	}

	if err := p.readVerack(); err != nil {
		fmt.Println("error reading verack message,", err)
		return err
	}

	return p.WriteMessage(nil, topics.VerAck)
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

	return p.WriteMessage(message, topics.Version)
}

func (p *Peer) readRemoteMsgVersion() error {
	topic, payload, err := p.readMessage()
	if err != nil {
		return err
	}

	if topic != topics.Version {
		return fmt.Errorf("Did not receive the expected '%s' message", topics.Version)
	}

	version, err := decodeVersionMessage(payload)
	if err != nil {
		return err
	}

	return p.verifyVersion(version)
}

func (p *Peer) readVerack() error {
	topic, _, err := p.readMessage()
	if err != nil {
		return err
	}

	if topic != topics.VerAck {
		return errors.New("read message was not a Verack message")
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
	versionMessage := &VersionMessage{}
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

func (p *Peer) verifyVersion(v *VersionMessage) error {
	if p.ProtoVer.Major != v.Version.Major {
		return errors.New("version mismatch")
	}

	return nil
}
