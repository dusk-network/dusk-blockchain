package peer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
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

	frame, err := processing.WriteFrame(fullMsg)
	if err != nil {
		return err
	}

	_, err = p.Write(frame.Bytes())
	return err
}

func (p *Connection) readRemoteMsgVersion() error {
	msgBytes, err := p.ReadMessage()
	if err != nil {
		return err
	}

	decodedMsg := new(bytes.Buffer)
	decodedMsg.Write(msgBytes)

	magic, err := extractMagic(decodedMsg)
	if err != nil {
		return err
	}

	if magic != p.magic {
		return errors.New("magic mismatch")
	}

	topic, err := extractTopic(decodedMsg)
	if err != nil {
		return err
	}

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

	decodedMsg := new(bytes.Buffer)
	decodedMsg.Write(msgBytes)

	magic, err := extractMagic(decodedMsg)
	if err != nil {
		return err
	}

	if magic != p.magic {
		return errors.New("magic mismatch")
	}

	topic, err := extractTopic(decodedMsg)
	if err != nil {
		return err
	}

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

	frame, err := processing.WriteFrame(verAckMsg)
	if err != nil {
		return err
	}

	if _, err := p.Write(frame.Bytes()); err != nil {
		return err
	}

	return nil
}

func (p *Connection) createVersionBuffer() (*bytes.Buffer, error) {
	version := protocol.NodeVer
	message, err := newVersionMessageBuffer(version, protocol.FullNode)
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
