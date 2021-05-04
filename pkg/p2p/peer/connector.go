// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package peer

import (
	"bytes"
	"context"
	"net"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	log "github.com/sirupsen/logrus"
)

// Connector is responsible for accepting incoming connection requests, and
// establishing outward connections with desired peers.
type Connector struct {
	l        net.Listener
	lock     sync.RWMutex
	registry map[string]struct{}
}

// NewConnector creates a new peer connector, and spawns a goroutine that will
// accept incoming connection requests on the current address, with the given port.
func NewConnector(port string) *Connector {
	addrPort := ":" + port

	listener, err := net.Listen("tcp", addrPort)
	if err != nil {
		log.WithField("process", "peer listener").
			WithError(err).
			Panic("could not establish a listener")
	}

	c := &Connector{l: listener, registry: make(map[string]struct{})}

	go func(c *Connector) {
		for {
			conn, err := c.l.Accept()
			if err != nil {
				log.WithField("process", "connection manager").
					WithError(err).
					Warnln("error accepting connection request")
				return
			}

			go c.acceptConnection(conn)
		}
	}(c)

	return c
}

func (c *Connector) Close() error {
	return c.l.Close()
}

// Connect dials a connection with its string, then on succession
// we pass the connection and the address to the OnConn method.
func (c *Connector) Connect(addr string) error {
	conn, err := c.Dial(addr)
	if err != nil {
		return err
	}

	go c.proposeConnection(conn)
	return nil
}

// Dial dials up a connection, given its address string.
func (c *Connector) Dial(addr string) (net.Conn, error) {
	dialTimeout := 1 * time.Second

	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (c *Connector) acceptConnection(conn net.Conn) {
	writeQueueChan := make(chan bytes.Buffer, 1000)

	peerReader := s.readerFactory.SpawnReader(conn, s.gossip, writeQueueChan)
	if err := peerReader.Accept(); err != nil {
		logServer.WithField("process", "peer connector").
			WithError(err).Warnln("problem performing incoming handshake")
		return
	}

	logServer.WithField("address", peerReader.Addr()).
		Debugln("incoming connection established")

	peerWriter := peer.NewWriter(conn, s.gossip, s.eventBus)

	c.addPeer(peerReader.Addr())
	go peer.Create(context.Background(), peerReader, peerWriter, writeQueueChan)
}

func (c *Connector) proposeConnection(conn net.Conn) {
	writeQueueChan := make(chan bytes.Buffer, 1000)
	peerWriter := peer.NewWriter(conn, s.gossip, s.eventBus)

	if err := peerWriter.Connect(); err != nil {
		logServer.WithField("process", "peer connector").
			WithError(err).Warnln("problem performing outgoing handshake")
		return
	}

	address := peerWriter.Addr()

	logServer.WithField("address", address).
		Debugln("outgoing connection established")

	peerReader := s.readerFactory.SpawnReader(conn, s.gossip, writeQueueChan)

	c.addPeer(peerWriter.Addr())
	go peer.Create(context.Background(), peerReader, peerWriter, writeQueueChan)
}

func (c *Connector) addPeer(address string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.registry[address] = struct{}{}
}

func (c *Connector) removePeer(address string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.registry, address)

	// Ensure we are still above the minimum connections threshold.
	if len(c.registry) < config.Get().Network.MinimumConnections {
		// Gossip address request to vouchers
	}
}