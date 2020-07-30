package peer

import (
	"bytes"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/checksum"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

var readWriteTimeout = 60 * time.Second // Max idle time for a peer
var keepAliveTime = 30 * time.Second    // Send keepalive message after inactivity for this amount of time

var l = log.WithField("process", "peer")

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
}

func (g *GossipConnector) Write(b []byte) (int, error) {
	buf := bytes.NewBuffer(b)
	if err := g.gossip.Process(buf); err != nil {
		return 0, err
	}

	return g.Connection.Write(buf.Bytes())
}

// Writer abstracts all of the logic and fields needed to write messages to
// other network nodes.
type Writer struct {
	*Connection
	subscriber eventbus.Subscriber
	gossipID   uint32
	keepAlive  time.Duration
	// TODO: add service flag
}

// Reader abstracts all of the logic and fields needed to receive messages from
// other network nodes.
type Reader struct {
	*Connection
	router   *messageRouter
	exitChan chan<- struct{} // Way to kill the WriteLoop
	// TODO: add service flag
}

// NewWriter returns a Writer. It will still need to be initialized by
// subscribing to the gossip topic with a stream handler, and by running the WriteLoop
// in a goroutine..
func NewWriter(conn net.Conn, gossip *processing.Gossip, subscriber eventbus.Subscriber, keepAlive ...time.Duration) *Writer {
	kas := 30 * time.Second
	if len(keepAlive) > 0 {
		kas = keepAlive[0]
	}
	pw := &Writer{
		Connection: &Connection{
			Conn:   conn,
			gossip: gossip,
		},
		subscriber: subscriber,
		keepAlive:  kas,
	}

	return pw
}

// NewReader returns a Reader. It will still need to be initialized by
// running ReadLoop in a goroutine.
func NewReader(conn net.Conn, gossip *processing.Gossip, dupeMap *dupemap.DupeMap, publisher eventbus.Publisher, rpcBus *rpcbus.RPCBus, counter *chainsync.Counter, responseChan chan<- *bytes.Buffer, exitChan chan<- struct{}) (*Reader, error) {
	pconn := &Connection{
		Conn:   conn,
		gossip: gossip,
	}

	_, db := heavy.CreateDBConnection()

	dataRequestor := responding.NewDataRequestor(db, rpcBus, responseChan)

	reader := &Reader{
		Connection: pconn,
		exitChan:   exitChan,
		router: &messageRouter{
			publisher:         publisher,
			dupeMap:           dupeMap,
			blockHashBroker:   responding.NewBlockHashBroker(db, responseChan),
			synchronizer:      chainsync.NewChainSynchronizer(publisher, rpcBus, responseChan, counter),
			dataRequestor:     dataRequestor,
			dataBroker:        responding.NewDataBroker(db, rpcBus, responseChan),
			roundResultBroker: responding.NewRoundResultBroker(rpcBus, responseChan),
			candidateBroker:   responding.NewCandidateBroker(rpcBus, responseChan),
			ponger:            processing.NewPonger(responseChan),
			peerInfo:          conn.RemoteAddr().String(),
		},
	}

	// On each new connection the node sends topics.Mempool to retrieve mempool
	// txs from the new peer
	go func() {
		if err := dataRequestor.RequestMempoolItems(); err != nil {
			l.WithError(err).Warnln("error sending topics.Mempool message")
		}
	}()

	return reader, nil
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
func (w *Writer) Connect() error {
	if err := w.Handshake(); err != nil {
		_ = w.Conn.Close()
		return err
	}

	return nil
}

// Accept will perform the protocol handshake with the peer.
func (p *Reader) Accept() error {
	if err := p.Handshake(); err != nil {
		_ = p.Conn.Close()
		return err
	}

	return nil
}

// Serve utilizes two different methods for writing to the open connection
func (w *Writer) Serve(writeQueueChan <-chan *bytes.Buffer, exitChan chan struct{}) {

	defer w.onDisconnect()

	// Any gossip topics are written into interrupt-driven ringBuffer
	// Single-consumer pushes messages to the socket
	g := &GossipConnector{w.gossip, w.Connection}
	w.gossipID = w.subscriber.Subscribe(topics.Gossip, eventbus.NewStreamListener(g))

	// writeQueue - FIFO queue
	// writeLoop pushes first-in message to the socket
	w.writeLoop(writeQueueChan, exitChan)
}

func (w *Writer) onDisconnect() {
	log.WithField("address", w.Connection.RemoteAddr().String()).Infof("Connection terminated")
	_ = w.Conn.Close()
	w.subscriber.Unsubscribe(topics.Gossip, w.gossipID)
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

// ReadLoop will block on the read until a message is read, or until the deadline
// is reached. Should be called in a go-routine, after a successful handshake with
// a peer. Eventual duplicated messages are silently discarded.
func (p *Reader) ReadLoop() {

	// As the peer ReadLoop is at the front-line of P2P network, receiving a
	// malformed frame by an adversary node could lead to a panic.
	// In such situation, the node should survive but adversary conn gets dropped
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Peer %s failed with critical issue: %v", p.RemoteAddr(), r)
		}
	}()

	p.readLoop()
}

func (p *Reader) readLoop() {
	defer func() {
		_ = p.Conn.Close()
	}()

	// Set up a timer, which triggers the sending of a `keepalive` message
	// when fired.
	timer, quitChan := p.keepAliveLoop()

	defer func() {
		p.exitChan <- struct{}{}
		quitChan <- struct{}{}
	}()

	for {
		// Refresh the read deadline
		err := p.Conn.SetReadDeadline(time.Now().Add(readWriteTimeout))
		if err != nil {
			l.WithError(err).Warnf("error setting read timeout")
		}

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

		// TODO: error here should be checked in order to decrease reputation
		// or blacklist spammers
		err = p.router.Collect(message)
		if err != nil {
			log.WithError(err).Errorln("error routing message")
		}

		// Reset the keepalive timer
		timer.Reset(keepAliveTime)
	}
}

func (p *Reader) keepAliveLoop() (*time.Timer, chan struct{}) {
	timer := time.NewTimer(keepAliveTime)
	quitChan := make(chan struct{}, 1)
	go func(p *Reader, t *time.Timer, quitChan chan struct{}) {
		for {
			select {
			case <-t.C:
				_ = p.Connection.keepAlive()
			case <-quitChan:
				t.Stop()
				return
			}
		}
	}(p, timer, quitChan)

	return timer, quitChan
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
	c.lock.Lock()
	_ = c.Conn.SetWriteDeadline(time.Now().Add(readWriteTimeout))
	n, err := c.Conn.Write(b)
	c.lock.Unlock()
	return n, err
}

// Addr returns the peer's address as a string.
func (c *Connection) Addr() string {
	return c.Conn.RemoteAddr().String()
}
