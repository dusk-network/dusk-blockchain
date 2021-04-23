// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package kadcast

import (
	"net"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// Reader is running a TCP listner handling TCP Broadcast packets from kadcast network
//
// On a TCP connection
// Read a single tcp packet from the accepted connection
// Validate the packet
// Extract Gossip Packet
// Build and Publish eventBus message
// The alternative to this reader is the RaptorCodeReader that is based on RC-UDP.
type Reader struct {
	base     *baseReader
	listener *net.TCPListener
}

// NewReader makes a new kadcast reader that handles TCP packets of broadcasting.
func NewReader(lpeerInfo encoding.PeerInfo, publisher eventbus.Publisher, gossip *protocol.Gossip, processor *peer.MessageProcessor) *Reader {
	addr := lpeerInfo.Address()

	lAddr, err := net.ResolveTCPAddr("tcp4", addr)
	if err != nil {
		log.Panicf("invalid kadcast peer address %s", addr)
	}

	l, err := net.ListenTCP("tcp4", lAddr)
	if err != nil {
		log.Panic(err)
	}

	r := new(Reader)
	r.base = newBaseReader(lpeerInfo, publisher, gossip, processor)
	r.listener = l

	log.WithField("l_addr", lAddr.String()).Infoln("Starting Reader")
	return r
}

// Close closes reader TCP listener.
func (r *Reader) Close() error {
	if r.listener != nil {
		return r.listener.Close()
	}

	return nil
}

// Serve starts accepting and processing TCP connection and packets.
func (r *Reader) Serve() {
	for {
		conn, err := r.listener.AcceptTCP()
		if err != nil {
			log.WithError(err).Warn("Error on tcp accept")
			return
		}

		// processPacket as for now a single-packet-per-connection is allowed
		go r.processPacket(conn)
	}
}

func (r *Reader) processPacket(conn *net.TCPConn) {
	// As the peer readloop is at the front-line of P2P network, receiving a
	// malformed frame by an adversary node could lead to a panic.
	defer func() {
		if r := recover(); r != nil {
			log.Error("readloop failed: ", r)
		}
	}()

	// Current impl expects only one TCPFrame per connection
	defer func() {
		if conn != nil {
			_ = conn.Close()
		}
	}()

	raddr := conn.RemoteAddr().String()

	// Read frame payload Set a new deadline for the connection.
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	b, err := readTCPFrame(conn)
	if err != nil {
		log.WithError(err).Warn("Error on frame read")
		return
	}

	_ = r.base.handleBroadcast(raddr, b)
}
