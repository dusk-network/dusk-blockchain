package kadcast

import (
	"errors"
	"fmt"
	"net"

	logger "github.com/sirupsen/logrus"
)

var log = logger.WithFields(logger.Fields{"process": "kadcast"})

// Maintainer is responsible for building and maintaining Kadcast routing state
type Maintainer struct {
	listener *net.UDPConn
	rtable   *RoutingTable
}

// NewMaintainer returns a UDP Reader for maintaining routing state up-to-date
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

// Close terminates udp read loop
// Even terminated, there might be some processPacket routines still running a job
func (m *Maintainer) Close() error {
	if m.listener != nil {
		return m.listener.Close()
	}
	return nil
}

// Serve starts maintainer main loop of listening and handling UDP packets
func (m *Maintainer) Serve() {

	for {
		b := make([]byte, MaxUDPacketSize)
		n, uAddr, err := m.listener.ReadFromUDP(b)
		if err != nil {
			log.WithError(err).Warn("Error on packet read")
			continue
		}

		go m.processPacket(*uAddr, b[0:n])
	}
}

func (m *Maintainer) processPacket(srcAddr net.UDPAddr, b []byte) {

	// As the peer readloop is at the front-line of P2P network, receiving a
	// malformed frame by an adversary node could lead to a panic.
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("readloop failed %v", r)
		}
	}()

	llog := log.WithField("l_addr", m.rtable.LpeerInfo.String()).WithField("r_addr", srcAddr.String())

	// Build packet struct
	var packet Packet
	port, msgType, err := unmarshalPacket(b, &packet, MaxUDPacketSize)
	if err != nil {
		llog.WithError(err).Warn("UDP reader rejects a packet")
		return
	}

	// Build Peer info and put the right port on it subsituting the one
	// used to send the message by the one where the peer wants to receive
	// the messages.
	ip := getPeerNetworkInfo(srcAddr)
	srcPeer := MakePeer(ip, port)

	llog.WithField("type", udpTipusToString(msgType)).
		WithField("len", len(b)).
		Traceln("Received UDP packet")

	// Check packet type and process it.
	var herr error
	switch msgType {
	case pingMsg:
		m.handlePing(srcPeer)
	case pongMsg:
		m.handlePong(srcPeer)
	case findNodesMsg:
		m.handleFindNodes(srcPeer)
	case nodesMsg:
		herr = m.handleNodes(srcPeer, packet, len(b))
	default:
		herr = fmt.Errorf("unknown msg type %d", msgType)
	}

	if herr != nil {
		llog.WithError(herr).Error("maintainer message handling failed")
	}
}

func (m *Maintainer) handlePing(peerInf PeerInfo) {

	rtable := m.rtable
	// Process peer addition to the tree.
	rtable.tree.addPeer(rtable.LpeerInfo, peerInf)
	// Send back a `PONG` message.
	rtable.sendPong(peerInf)
}

func (m *Maintainer) handlePong(peerInf PeerInfo) {
	rtable := m.rtable
	// Process peer addition to the tree.
	rtable.tree.addPeer(rtable.LpeerInfo, peerInf)
}

func (m *Maintainer) handleFindNodes(peerInf PeerInfo) {
	rtable := m.rtable
	// Process peer addition to the tree.
	rtable.tree.addPeer(rtable.LpeerInfo, peerInf)
	// Send back a `NODES` message to the peer that
	// send the `FIND_NODES` message.
	rtable.sendNodes(peerInf)
}

func (m *Maintainer) handleNodes(peerInf PeerInfo, packet Packet, byteNum int) error {

	// See if the packet info is consistent:
	// peerNum announced <=> bytesPerPeer * peerNum
	if !packet.checkNodesPayloadConsistency(byteNum) {
		// Since the packet is not consistent, we just discard it.
		return errors.New("NODES message rejected due to corrupted payload")
	}

	rtable := m.rtable
	// Process peer addition to the tree.
	rtable.tree.addPeer(rtable.LpeerInfo, peerInf)

	// Deserialize the payload to get the peerInfo of every
	// received peer.
	peers := packet.getNodesPayloadInfo()

	// Send `PING` messages to all of the peers to then
	// add them to our buckets if they respond with a `PONG`.
	for _, peer := range peers {
		rtable.sendPing(peer)
	}

	return nil
}
