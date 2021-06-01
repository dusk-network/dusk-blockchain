// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast/encoding"
	"github.com/sirupsen/logrus"
	logger "github.com/sirupsen/logrus"
)

var log = logger.WithFields(logger.Fields{"process": "kadcast"})

// Maintainer is responsible for building and maintaining Kadcast routing state.
type Maintainer struct {
	listener *net.UDPConn
	rtable   *RoutingTable
}

// NewMaintainer returns a UDP Reader for maintaining routing state up-to-date.
func NewMaintainer(rtable *RoutingTable) *Maintainer {
	if rtable == nil {
		log.Panic("cannot launch with nil routing table")
	}

	addr := rtable.LpeerInfo.Address()

	lAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		log.Panicf("invalid kadcast peer address %s", addr)
	}

	listener, err := net.ListenUDP("udp4", lAddr)
	if err != nil {
		log.Panic(err)
	}

	m := Maintainer{
		listener: listener,
		rtable:   rtable,
	}

	log.WithField("l_addr", lAddr.String()).Infof("Starting Routing-state Maintainer")

	return &m
}

// Close terminates udp read loop.
// Even terminated, there might be some processPacket routines still running a job.
func (m *Maintainer) Close() error {
	if m.listener != nil {
		return m.listener.Close()
	}

	return nil
}

// Serve starts maintainer main loop of listening and handling UDP packets.
func (m *Maintainer) Serve() {
	for {
		b := make([]byte, MaxUDPacketSize)

		n, uAddr, err := m.listener.ReadFromUDP(b)
		if err != nil {
			log.WithError(err).Warn("Error on packet read")
			continue
		}

		data := bytes.NewBuffer(b[0:n])

		go m.processPacket(*uAddr, data)
	}
}

func (m *Maintainer) processPacket(srcAddr net.UDPAddr, buf *bytes.Buffer) {
	// As the peer readloop is at the front-line of P2P network, receiving a
	// malformed frame by an adversary node could lead to a panic.
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("readloop failed %v", r)
		}
	}()

	lAddr := m.rtable.LpeerInfo.String()
	rAddr := srcAddr.String()
	llog := log.WithField("l_addr", lAddr).WithField("r_addr", rAddr)

	var header encoding.Header
	if err := header.UnmarshalBinary(buf); err != nil {
		return
	}

	// Make remote peerInfo based on addr from IP datagram and RemotePeerPort
	// from header
	addr := srcAddr.String()

	remotePeer, err := encoding.MakePeerFromIP(addr, header.RemotePeerPort)
	if err != nil {
		llog.WithError(err).Warn("TCP reader found invalid IP")
		return
	}

	// Ensure the RemotePeerID from header is correct one
	// This together with Nonce-PoW aims at providing a bit of DDoS protection
	if !bytes.Equal(remotePeer.ID[:], header.RemotePeerID[:]) {
		llog.Error("Invalid remote peer id")
		return
	}

	tn, _ := encoding.MsgTypeToString(header.MsgType)

	llog.WithField("msg_type", tn).WithField("msg_len", buf.Len()).
		Traceln("Received UDP packet")

	// Check packet type and process it.
	err = nil

	switch header.MsgType {
	case encoding.PingMsg:
		err = m.handlePing(remotePeer)
	case encoding.PongMsg:
		m.handlePong(remotePeer)
	case encoding.FindNodesMsg:
		err = m.handleFindNodes(remotePeer)
	case encoding.NodesMsg:
		var p encoding.NodesPayload
		err = p.UnmarshalBinary(buf)

		if err == nil {
			m.handleNodes(remotePeer, p.Peers)
		}

	default:
		err = fmt.Errorf("unknown message type id %d", header.MsgType)
	}

	if err != nil {
		llog.WithError(err).Error("maintainer message handling failed")
	}
}

func (m *Maintainer) handlePing(peerInf encoding.PeerInfo) error {
	rtable := m.rtable

	// Process peer addition to the tree.
	rtable.tree.addPeer(rtable.LpeerInfo, peerInf)

	// Send back a `PONG` message.
	return m.sendPong(peerInf)
}

func (m *Maintainer) handlePong(peerInf encoding.PeerInfo) {
	rtable := m.rtable
	// Process peer addition to the tree.

	rtable.tree.addPeer(rtable.LpeerInfo, peerInf)
}

func (m *Maintainer) handleFindNodes(peerInf encoding.PeerInfo) error {
	rtable := m.rtable

	// Register sending peer
	rtable.tree.addPeer(rtable.LpeerInfo, peerInf)

	// Respond with set of nodes
	return m.sendNodesMsg(peerInf)
}

func (m *Maintainer) handleNodes(peerInf encoding.PeerInfo, peers []encoding.PeerInfo) {
	rtable := m.rtable
	// Process peer addition to the tree.

	rtable.tree.addPeer(rtable.LpeerInfo, peerInf)

	for _, peer := range peers {
		_ = m.sendPing(peer)
	}
}

func (m *Maintainer) sendNodesMsg(receiver encoding.PeerInfo) error {
	// Get `K` closest peers to `targetPeer`
	kClosestPeers := m.rtable.getXClosestPeersTo(DefaultKNumber, receiver)
	if len(kClosestPeers) == 0 {
		log.Tracef("could not get closest peers for remote peer %s", receiver.String())
		return nil
	}

	h := makeHeader(encoding.NodesMsg, m.rtable)
	p := encoding.NodesPayload{Peers: kClosestPeers}

	var buf bytes.Buffer
	if err := encoding.MarshalBinary(h, &p, &buf); err != nil {
		return err
	}

	m.send(receiver.GetUDPAddr(), buf.Bytes())
	return nil
}

func (m *Maintainer) sendPong(receiver encoding.PeerInfo) error {
	h := makeHeader(encoding.PongMsg, m.rtable)

	var buf bytes.Buffer
	if err := encoding.MarshalBinary(h, nil, &buf); err != nil {
		return err
	}

	m.send(receiver.GetUDPAddr(), buf.Bytes())
	return nil
}

func (m *Maintainer) sendPing(receiver encoding.PeerInfo) error {
	h := makeHeader(encoding.PingMsg, m.rtable)

	var buf bytes.Buffer
	if err := encoding.MarshalBinary(h, nil, &buf); err != nil {
		return err
	}

	m.send(receiver.GetUDPAddr(), buf.Bytes())
	return nil
}

func (m *Maintainer) send(raddr net.UDPAddr, payload []byte) {
	laddr := m.rtable.lpeerUDPAddr

	log.WithField("dest", raddr.String()).Tracef("Dialing udp")

	// Send from same IP that the UDP listener is bound on but choose random port
	laddr.Port = 0

	conn, err := net.DialUDP("udp", &laddr, &raddr)
	if err != nil {
		log.WithError(err).Warn("Could not establish a connection with the dest Peer.")
		return
	}

	defer func() {
		_ = conn.Close()
	}()

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	if log.Logger.GetLevel() == logrus.TraceLevel {
		log.WithField("src", conn.LocalAddr().String()).
			WithField("dest", raddr.String()).Traceln("Sending udp")
	}

	// Simple write
	_, err = conn.Write(payload)
	if err != nil {
		log.WithError(err).Warn("Error while writing to the filedescriptor.")
		return
	}
}
