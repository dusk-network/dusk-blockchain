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
func (w *Writer) Handshake(services protocol.ServiceFlag) error {
	if err := w.writeLocalMsgVersion(w.gossip, services); err != nil {
		return err
	}

	if err := w.readVerAck(); err != nil {
		return err
	}

	peerServices, err := w.readRemoteMsgVersion()
	if err != nil {
		return err
	}

	w.services = peerServices
	return w.writeVerAck(w.gossip)
}

// Handshake with another peer.
func (p *Reader) Handshake(services protocol.ServiceFlag) error {
	peerServices, err := p.readRemoteMsgVersion()
	if err != nil {
		return err
	}

	p.services = peerServices

	if err := p.writeVerAck(p.gossip); err != nil {
		return err
	}

	if err := p.writeLocalMsgVersion(p.gossip, services); err != nil {
		return err
	}

	return p.readVerAck()
}

func (c *Connection) writeLocalMsgVersion(g *protocol.Gossip, services protocol.ServiceFlag) error {
	message, e := c.createVersionBuffer(services)
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

func (c *Connection) readRemoteMsgVersion() (protocol.ServiceFlag, error) {
	msgBytes, err := c.ReadMessage()
	if err != nil {
		return 0, err
	}

	m, cs, err := checksum.Extract(msgBytes)
	if err != nil {
		return 0, err
	}

	if !checksum.Verify(m, cs) {
		return 0, errors.New("invalid checksum")
	}

	decodedMsg := bytes.NewBuffer(m)

	topic, err := topics.Extract(decodedMsg)
	if err != nil {
		return 0, err
	}

	if topic != topics.Version {
		return 0, fmt.Errorf("did not receive the expected '%s' message - got %s",
			topics.Version, topic)
	}

	version, err := decodeVersionMessage(decodedMsg)
	if err != nil {
		return 0, err
	}

	return version.Services, verifyVersionMessage(version)
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

func (c *Connection) createVersionBuffer(services protocol.ServiceFlag) (*bytes.Buffer, error) {
	version := protocol.NodeVer

	message, err := newVersionMessageBuffer(version, services)
	if err != nil {
		return nil, err
	}

	return message, nil
}

func verifyVersionMessage(v *VersionMessage) error {
	if protocol.NodeVer.Major != v.Version.Major {
		return errors.New("version mismatch")
	}

	if v.Services != protocol.FullNode && v.Services != protocol.VoucherNode {
		return errors.New("unknown service flag")
	}

	return nil
}
