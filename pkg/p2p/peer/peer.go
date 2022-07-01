// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package peer

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/capi"

	log "github.com/sirupsen/logrus"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/checksum"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/container/ring"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// Filtering logs by "peer" or "peer_conn" words should be enough to fetch all
// details of gossip connections and messages for all peers.
var l = log.WithField("process", "peer")

const (
	defaultTimeoutReadWrite = 60
	defaultKeepAliveTime    = 30
)

// Connection holds the TCP connection to another node, and it's known protocol magic.
// The `net.Conn` is guarded by a mutex, to allow both multicast and one-to-one
// communication between peers.
type Connection struct {
	lock sync.Mutex
	net.Conn
	gossip   *protocol.Gossip
	services protocol.ServiceFlag //nolint:structcheck
}

// NewConnection creates a peer connection struct.
func NewConnection(conn net.Conn, gossip *protocol.Gossip) *Connection {
	return &Connection{
		Conn:   conn,
		gossip: gossip,
	}
}

// GossipConnector calls Gossip.Process on the message stream incoming from the
// ringbuffer.
// It absolves the function previously carried over by the Gossip preprocessor.
type GossipConnector struct {
	*Connection
}

func (g *GossipConnector) Write(b, header []byte, priority byte) (int, error) {
	if !canRoute(g.services, topics.Topic(b[0])) {
		return 0, nil
	}

	buf := bytes.NewBuffer(b)
	if err := g.gossip.Process(buf); err != nil {
		return 0, err
	}

	n, err := g.Connection.Write(buf.Bytes())
	if err != nil {
		l.WithField("r_addr", g.RemoteAddr().String()).
			WithError(err).Warn("failed to write")
	}

	return n, err
}

// Writer abstracts all of the logic and fields needed to write messages to
// other network nodes.
type Writer struct {
	*Connection
	subscriber eventbus.Subscriber
	gossipID   uint32
	keepAlive  time.Duration
}

// Reader abstracts all of the logic and fields needed to receive messages from
// other network nodes.
type Reader struct {
	*Connection
	processor *MessageProcessor
}

// NewWriter returns a Writer. It will still need to be initialized by
// subscribing to the gossip topic with a stream handler, and by running the WriteLoop
// in a goroutine.
func NewWriter(conn *Connection, subscriber eventbus.Subscriber, keepAlive ...time.Duration) *Writer {
	kas := 30 * time.Second
	if len(keepAlive) > 0 {
		kas = keepAlive[0]
	}

	pw := &Writer{
		Connection: conn,
		subscriber: subscriber,
		keepAlive:  kas,
	}

	return pw
}

// ReadMessage reads from the connection.
func (c *Connection) ReadMessage() ([]byte, error) {
	length, err := c.gossip.UnpackLength(c.Conn)
	if err != nil {
		return nil, err
	}

	// read a [length]byte from connection
	buf := make([]byte, int(length))

	_, err = io.ReadFull(c.Conn, buf)
	if err != nil {
		return nil, err
	}

	return buf, err
}

// Connect will perform the protocol handshake with the peer. If successful...
func (w *Writer) Connect(services protocol.ServiceFlag) error {
	if err := w.Handshake(services); err != nil {
		_ = w.Conn.Close()
		return err
	}

	if config.Get().API.Enabled {
		go func() {
			store := capi.GetStormDBInstance()
			addr := w.Addr()
			peerJSON := capi.PeerJSON{
				Address:  addr,
				Type:     "Writer",
				Method:   "Connect",
				LastSeen: time.Now(),
			}

			err := store.Save(&peerJSON)
			if err != nil {
				l.Error("failed to save peerJSON into StormDB")
			}

			// save count
			peerCount := capi.PeerCount{
				ID:       addr,
				LastSeen: time.Now(),
			}

			err = store.Save(&peerCount)
			if err != nil {
				l.Error("failed to save peerCount into StormDB")
			}
		}()
	}

	return nil
}

// Accept will perform the protocol handshake with the peer.
func (p *Reader) Accept(services protocol.ServiceFlag) error {
	if err := p.Handshake(services); err != nil {
		_ = p.Conn.Close()
		return err
	}

	if config.Get().API.Enabled {
		go func() {
			store := capi.GetStormDBInstance()
			addr := p.Addr()
			peerJSON := capi.PeerJSON{
				Address:  addr,
				Type:     "Reader",
				Method:   "Accept",
				LastSeen: time.Now(),
			}

			err := store.Save(&peerJSON)
			if err != nil {
				l.Error("failed to save peer into StormDB")
			}

			// save count
			peerCount := capi.PeerCount{
				ID:       addr,
				LastSeen: time.Now(),
			}

			err = store.Save(&peerCount)
			if err != nil {
				l.Error("failed to save peerCount into StormDB")
			}
		}()
	}

	return nil
}

// Create two-way communication with a peer. This function will allow both
// goroutines to run as long as no errors are encountered. Once the first error
// comes through, the context is canceled, and both goroutines are cleaned up.
func Create(ctx context.Context, reader *Reader, writer *Writer) {
	pCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	g := &GossipConnector{writer.Connection}
	listener := eventbus.NewStreamListener(g)
	writer.gossipID = writer.subscriber.Subscribe(topics.Gossip, listener)
	ringBuf := ring.NewBuffer(1000)

	// On each new connection the node sends topics.Mempool to retrieve mempool
	// txs from the new peer
	buf := topics.MemPool.ToBuffer()

	e := ring.Elem{
		Data: buf.Bytes(),
	}

	if !ringBuf.Put(e) {
		l.Errorln("could not send mempool message to peer")
	}

	_ = ring.NewConsumer(ringBuf, eventbus.Consume, g, false)

	reader.ReadLoop(pCtx, ringBuf)
	writer.onDisconnect()
}

func (w *Writer) onDisconnect() {
	l.WithField("r_addr", w.Connection.RemoteAddr().String()).Info("peer_connection terminated")

	_ = w.Conn.Close()

	w.subscriber.Unsubscribe(topics.Gossip, w.gossipID)

	if config.Get().API.Enabled {
		go func() {
			store := capi.GetStormDBInstance()
			addr := w.Addr()
			peerJSON := capi.PeerJSON{
				Address:  addr,
				Type:     "Writer",
				Method:   "onDisconnect",
				LastSeen: time.Now(),
			}

			err := store.Save(&peerJSON)
			if err != nil {
				l.Error("failed to save peer into StormDB")
			}
		}()
	}
}

// ReadLoop will block on the read until a message is read, or until the deadline
// is reached. Should be called in a go-routine, after a successful handshake with
// a peer. Eventual duplicated messages are silently discarded.
func (p *Reader) ReadLoop(ctx context.Context, ringBuf *ring.Buffer) {
	defer func() {
		_ = p.Conn.Close()
	}()

	trw := defaultTimeoutReadWrite

	readWriteTimeout := time.Duration(trw) * time.Second

	// Set up a timer, which triggers the sending of a `keepalive` message
	// when fired.
	kat := defaultKeepAliveTime

	keepAliveTime := time.Duration(kat) * time.Second

	timer := time.NewTimer(keepAliveTime)
	go p.keepAliveLoop(ctx, timer)

	plog := l.WithField("r_addr", p.Conn.RemoteAddr().String())

	for {
		// Check if context was canceled
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Refresh the read deadline
		err := p.Conn.SetReadDeadline(time.Now().Add(readWriteTimeout))
		if err != nil {
			plog.WithError(err).
				Warn("error setting read timeout")
			return
		}

		b, err := p.gossip.ReadMessage(p.Conn)
		if err != nil {
			plog.WithError(err).Warnln("error reading message")
			return
		}

		message, cs, err := checksum.Extract(b)
		if err != nil {
			plog.WithError(err).Warnln("error extracting message and cs")
			return
		}

		if !checksum.Verify(message, cs) {
			plog.WithError(errors.New("invalid checksum")).
				Warnln("error verifying message cs")
			return
		}

		go func() {
			// TODO: error here should be checked in order to decrease reputation
			// or blacklist spammers
			if _, err = p.processor.Collect(p.Addr(), message, ringBuf, p.services, nil); err != nil {
				var topic string
				if len(message) > 0 {
					topic = topics.Topic(message[0]).String()
				}

				plog.WithField("cs", hex.EncodeToString(cs)).
					WithField("topic", topic).
					WithError(err).Error("failed to process message")
			}
		}()

		// Reset the keepalive timer
		timer.Reset(keepAliveTime)
	}
}

func (p *Reader) keepAliveLoop(ctx context.Context, timer *time.Timer) {
	for {
		select {
		case <-timer.C:
			if err := p.Connection.keepAlive(); err != nil {
				l.WithError(err).WithField("r_addr", p.RemoteAddr().String()).
					Error("failed to ping a peer")
			}
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

func (c *Connection) keepAlive() error {
	buf := new(bytes.Buffer)
	if err := topics.Prepend(buf, topics.Ping); err != nil {
		return err
	}

	if err := c.gossip.Process(buf); err != nil {
		return err
	}

	_, err := c.Write(buf.Bytes())
	return err
}

// Write a message to the connection.
// Conn needs to be locked, as this function can be called both by the WriteLoop,
// and by the writer on the ring buffer.
func (c *Connection) Write(b []byte) (int, error) {
	wt := 1

	writeTimeout := time.Duration(wt) * time.Second // Max idle time for a peer

	c.lock.Lock()
	_ = c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	n, err := c.Conn.Write(b)
	c.lock.Unlock()

	return n, err
}

// Addr returns the peer's address as a string.
func (c *Connection) Addr() string {
	return c.Conn.RemoteAddr().String()
}
