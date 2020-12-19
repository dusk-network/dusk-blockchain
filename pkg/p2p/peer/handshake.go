// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package peer

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/checksum"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// Handshake with another peer.
func (w *Writer) Handshake() error {
	if err := w.writeLocalMsgVersion(w.gossip); err != nil {
		return err
	}

	if err := w.readVerAck(); err != nil {
		return err
	}

	if err := w.readRemoteMsgVersion(); err != nil {
		return err
	}

	return w.writeVerAck(w.gossip)
}

// Handshake with another peer.
func (p *Reader) Handshake() error {
	if err := p.readRemoteMsgVersion(); err != nil {
		return err
	}

	if err := p.writeVerAck(p.gossip); err != nil {
		return err
	}

	if err := p.writeLocalMsgVersion(p.gossip); err != nil {
		return err
	}

	return p.readVerAck()
}

func (c *Connection) writeLocalMsgVersion(g *protocol.Gossip) error {
	message, e := c.createVersionBuffer()
	if e != nil {
		return e
	}

	if err := topics.Prepend(message, topics.Version); err != nil {
		return err
	}

	if err := g.Process(message); err != nil {
		return err
	}

	_, e = c.Write(message.Bytes())
	return e
}

func (c *Connection) readRemoteMsgVersion() error {
	msgBytes, err := c.ReadMessage()
	if err != nil {
		return err
	}

	m, cs, err := checksum.Extract(msgBytes)
	if err != nil {
		return err
	}

	if !checksum.Verify(m, cs) {
		return errors.New("invalid checksum")
	}

	decodedMsg := bytes.NewBuffer(m)
	topic, err := topics.Extract(decodedMsg)
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

func (c *Connection) readVerAck() error {
	msgBytes, err := c.ReadMessage()
	if err != nil {
		return err
	}

	m, cs, err := checksum.Extract(msgBytes)
	if err != nil {
		return err
	}

	if !checksum.Verify(m, cs) {
		return errors.New("invalid checksum")
	}

	decodedMsg := bytes.NewBuffer(m)
	topic, err := topics.Extract(decodedMsg)
	if err != nil {
		return err
	}

	if topic != topics.VerAck {
		return fmt.Errorf("did not receive the expected '%s' message - got %s",
			topics.VerAck, topic)
	}

	return nil
}

func (c *Connection) writeVerAck(g *protocol.Gossip) error {
	verAckMsg := new(bytes.Buffer)
	if err := topics.Prepend(verAckMsg, topics.VerAck); err != nil {
		return err
	}

	if err := g.Process(verAckMsg); err != nil {
		return err
	}

	if _, err := c.Write(verAckMsg.Bytes()); err != nil {
		return err
	}

	return nil
}

func (c *Connection) createVersionBuffer() (*bytes.Buffer, error) {
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
