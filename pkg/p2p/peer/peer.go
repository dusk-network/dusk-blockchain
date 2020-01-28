package peer

import (
	"bytes"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/checksum"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

var readWriteTimeout = 60 * time.Second // Max idle time for a peer

var l *log.Entry = log.WithField("process", "peer")

// Router is an abstraction over the message router used by a
// peer.Reader.
type Router interface {
	// Collect a message and route it to the proper component.
	Collect(*bytes.Buffer) error
}

// Connection holds the TCP connection to another node, and it's known protocol magic.
// The `net.Conn` is guarded by a mutex, to allow both multicast and one-to-one
// communication between peers.
type Connection struct {
	lock sync.Mutex
	net.Conn
	gossip *processing.Gossip
}

// GossipConnector calls Gossip.Process on the message stream incoming from the
// ringbuffer
// It absolves the function previously carried over by the Gossip preprocessor.
type GossipConnector struct {
	gossip *processing.Gossip
	*Connection
	protocol.ServiceFlag
}

func (g *GossipConnector) Write(b []byte) (int, error) {
	// Preliminary check, to ensure we don't send messages to
	// nodes who aren't interested in them.
	if !g.canSend(b[0]) {
		return 0, nil
	}

	buf := bytes.NewBuffer(b)
	if err := g.gossip.Process(buf); err != nil {
		return 0, err
	}

	return g.Connection.Write(buf.Bytes())
}

func (g *GossipConnector) canSend(topicByte byte) bool {
	if g.ServiceFlag == protocol.FullNode {
		return true
	}

	switch topics.Topic(topicByte) {
	case topics.Block, topics.Inv:
		return true
	default:
		return false
	}
}

// Writer abstracts all of the logic and fields needed to write messages to
// other network nodes.
type Writer struct {
	*Connection
	subscriber eventbus.Subscriber
	gossipID   uint32
}

// Reader abstracts all of the logic and fields needed to receive messages from
// other network nodes.
type Reader struct {
	*Connection
	router   Router
	exitChan chan<- struct{} // Way to kill the WriteLoop
}

// NewWriter returns a Writer. It will still need to be initialized by
// subscribing to the gossip topic with a stream handler, and by running the WriteLoop
// in a goroutine..
func NewWriter(conn net.Conn, gossip *processing.Gossip, subscriber eventbus.Subscriber) *Writer {
	pw := &Writer{
		Connection: &Connection{
			Conn:   conn,
			gossip: gossip,
		},
		subscriber: subscriber,
	}

	return pw
}

// NewReader returns a Reader. It will still need to be initialized by
// running ReadLoop in a goroutine.
func NewReader(conn net.Conn, gossip *processing.Gossip, exitChan chan<- struct{}) *Reader {
	return &Reader{
		Connection: &Connection{
			Conn:   conn,
			gossip: gossip,
		},
		exitChan: exitChan,
	}
}

// ReadMessage reads from the connection
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

// Connect will perform the protocol handshake with the peer. If successful
func (p *Writer) Connect() (protocol.ServiceFlag, error) {
	serviceFlag, err := p.Handshake()
	if err != nil {
		p.Conn.Close()
		return 0, err
	}

	return serviceFlag, nil
}

// Accept will perform the protocol handshake with the peer.
func (p *Reader) Accept() (protocol.ServiceFlag, error) {
	serviceFlag, err := p.Handshake()
	if err != nil {
		p.Conn.Close()
		return 0, err
	}

	return serviceFlag, nil
}

// Serve utilizes two different methods for writing to the open connection
func (w *Writer) Serve(writeQueueChan <-chan *bytes.Buffer, exitChan chan struct{}, serviceFlag protocol.ServiceFlag) {
	defer w.onDisconnect()

	// Any gossip topics are written into interrupt-driven ringBuffer
	// Single-consumer pushes messages to the socket
	g := &GossipConnector{w.gossip, w.Connection, serviceFlag}
	w.gossipID = w.subscriber.Subscribe(topics.Gossip, eventbus.NewStreamListener(g))

	// Ping loop - ensures connection stays alive during quiet periods
	go w.pingLoop()

	// writeQueue - FIFO queue
	// writeLoop pushes first-in message to the socket
	w.writeLoop(writeQueueChan, exitChan)
}

func (w *Writer) onDisconnect() {
	log.Infof("Connection to %s terminated", w.Connection.RemoteAddr().String())
	w.Conn.Close()
	w.subscriber.Unsubscribe(topics.Gossip, w.gossipID)
}

func (w *Writer) pingLoop() {
	// We ping every 30 seconds to keep the connection alive
	ticker := time.NewTicker(30 * time.Second)

	for {
		<-ticker.C
		if err := w.ping(); err != nil {
			l.WithError(err).Warnln("error pinging peer")
			// Clean up ticker
			ticker.Stop()
			return
		}
	}
}

func (w *Writer) ping() error {
	buf := new(bytes.Buffer)
	if err := topics.Prepend(buf, topics.Ping); err != nil {
		return err
	}

	if err := w.gossip.Process(buf); err != nil {
		return err
	}

	_, err := w.Connection.Write(buf.Bytes())
	return err
}

func (w *Writer) writeLoop(writeQueueChan <-chan *bytes.Buffer, exitChan chan struct{}) {

	for {
		select {
		case buf := <-writeQueueChan:
			if err := w.gossip.Process(buf); err != nil {
				l.WithError(err).Warnln("error processing outgoing message")
				continue
			}

			if _, err := w.Connection.Write(buf.Bytes()); err != nil {
				l.WithField("queue", "writequeue").WithError(err).Warnln("error writing message")
				exitChan <- struct{}{}
			}
		case <-exitChan:
			return
		}
	}
}

// Listen will set up the correct router for the peer, and starts
// listening for messages.
// Should be called in a go-routine, after a successful handshake with
// a peer.
func (p *Reader) Listen(publisher eventbus.Publisher, dupeMap *dupemap.DupeMap, rpcBus *rpcbus.RPCBus, counter *chainsync.Counter, responseChan chan<- *bytes.Buffer, serviceFlag protocol.ServiceFlag) {
	_, db := heavy.CreateDBConnection()
	if !config.Get().General.LightNode {
		router := newRouter(publisher, dupeMap, db, rpcBus, counter, responseChan, p.Conn.RemoteAddr().String(), serviceFlag)
		p.router = router

		// On each new connection the node sends topics.Mempool to
		// retrieve mempool txs from the new peer
		if serviceFlag == protocol.FullNode {
			go func() {
				if err := router.dataRequestor.RequestMempoolItems(); err != nil {
					l.WithError(err).Warnln("error sending topics.Mempool message")
				}
			}()
		}
	} else {
		p.router = newLightRouter(publisher, db, rpcBus, counter, responseChan, p.Conn.RemoteAddr().String())
	}

	p.ReadLoop()
}

// ReadLoop will block on the read until a message is read, or until the deadline
// is reached. Eventual duplicated messages are silently discarded.
func (p *Reader) ReadLoop() {
	defer p.Conn.Close()
	defer func() {
		p.exitChan <- struct{}{}
	}()

	for {
		// Refresh the read deadline
		p.Conn.SetReadDeadline(time.Now().Add(readWriteTimeout))

		b, err := p.ReadMessage()
		if err != nil {
			l.WithError(err).Warnln("error reading message")
			return
		}

		message, cs, err := checksum.Extract(b)
		if err != nil {
			l.WithError(err).Warnln("error reading message")
			return
		}

		if !checksum.Verify(message, cs) {
			l.WithError(errors.New("invalid checksum")).Warnln("error reading message")
			return
		}

		p.router.Collect(bytes.NewBuffer(message))
	}
}

// Write a message to the connection.
// Conn needs to be locked, as this function can be called both by the WriteLoop,
// and by the writer on the ring buffer.
func (c *Connection) Write(b []byte) (int, error) {
	c.lock.Lock()
	c.Conn.SetWriteDeadline(time.Now().Add(readWriteTimeout))
	n, err := c.Conn.Write(b)
	c.lock.Unlock()
	return n, err
}

// Addr returns the peer's address as a string.
func (c *Connection) Addr() string {
	return c.Conn.RemoteAddr().String()
}
