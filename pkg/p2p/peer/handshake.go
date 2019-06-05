package peer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/util"
)

// Handshake with another peer.
func (p *Writer) Handshake() error {
	if err := p.writeLocalMsgVersion(); err != nil {
		return err
	}

	if err := p.readVerAck(); err != nil {
		return err
	}

	if err := p.readRemoteMsgVersion(); err != nil {
		return err
	}

	return p.writeVerAck()
}

// Handshake with another peer.
func (p *Reader) Handshake() error {
	if err := p.readRemoteMsgVersion(); err != nil {
		return err
	}

	if err := p.writeVerAck(); err != nil {
		return err
	}

	if err := p.writeLocalMsgVersion(); err != nil {
		return err
	}

	return p.readVerAck()
}

func (p *Connection) writeLocalMsgVersion() error {
	message, err := p.createVersionBuffer()
	if err != nil {
		return err
	}

	fullMsg, err := p.addHeader(message, topics.Version)
	if err != nil {
		return err
	}

	encodedMsg := processing.Encode(fullMsg).Bytes()
	_, err = p.Conn.Write(encodedMsg)
	return err
}

func (p *Connection) readRemoteMsgVersion() error {
	msgBytes, err := p.ReadMessage()
	if err != nil {
		return err
	}

	decodedMsg := processing.Decode(msgBytes)
	magic := extractMagic(decodedMsg)
	if magic != p.magic {
		return errors.New("magic mismatch")
	}

	topic := extractTopic(decodedMsg)
	if topic != topics.Version {
		return fmt.Errorf("did not receive the expected '%s' message - got %s",
			topics.Version, topic)
	}

	version, err := decodeVersionMessage(decodedMsg)
	if err != nil {
		return err
	}

	return verifyVersion(version.Version)
}

func (p *Connection) addHeader(m *bytes.Buffer, topic topics.Topic) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := encoding.WriteUint32(buf, binary.LittleEndian, uint32(p.magic)); err != nil {
		return nil, err
	}

	topicBytes := topics.TopicToByteArray(topic)
	if _, err := buf.Write(topicBytes[:]); err != nil {
		return nil, err
	}

	if _, err := buf.ReadFrom(m); err != nil {
		return nil, err
	}

	return buf, nil
}

func (p *Connection) readVerAck() error {
	msgBytes, err := p.ReadMessage()
	if err != nil {
		return err
	}

	decodedMsg := processing.Decode(msgBytes)
	magic := extractMagic(decodedMsg)
	if magic != p.magic {
		return errors.New("magic mismatch")
	}

	topic := extractTopic(decodedMsg)
	if topic != topics.VerAck {
		return fmt.Errorf("did not receive the expected '%s' message - got %s",
			topics.VerAck, topic)
	}

	return nil
}

func (p *Connection) writeVerAck() error {
	verAckMsg, err := p.addHeader(new(bytes.Buffer), topics.VerAck)
	if err != nil {
		return err
	}

	encodedMsg := processing.Encode(verAckMsg)
	if _, err := p.Conn.Write(encodedMsg.Bytes()); err != nil {
		return err
	}

	return nil
}

func (p *Connection) createVersionBuffer() (*bytes.Buffer, error) {
	fromPort := uint16(p.Port())
	version := protocol.NodeVer
	localIP, err := util.GetOutboundIP()
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return message, nil
}

func verifyVersion(v *protocol.Version) error {
	if protocol.NodeVer.Major != v.Major {
		return errors.New("version mismatch")
	}

	return nil
}
