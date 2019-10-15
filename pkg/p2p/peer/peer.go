package peer

import (
	"bytes"
	"io"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

var readWriteTimeout = 60 * time.Second // Max idle time for a peer

// Connection holds the TCP connection to another node, and it's known protocol magic.
// The `net.Conn` is guarded by a mutex, to allow both multicast and one-to-one
// communication between peers.
type Connection struct {
	lock sync.Mutex
	net.Conn
	gossip *processing.Gossip
}

// Writer abstracts all of the logic and fields needed to write messages to
// other network nodes.
type Writer struct {
	*Connection
	subscriber eventbus.Subscriber
	gossipID   uint32
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
func NewReader(conn net.Conn, gossip *processing.Gossip, dupeMap *dupemap.DupeMap, publisher eventbus.Publisher, rpcBus *rpcbus.RPCBus, counter *chainsync.Counter, responseChan chan<- *bytes.Buffer, exitChan chan<- struct{}) (*Reader, error) {
	pconn := &Connection{
		Conn:   conn,
		gossip: gossip,
	}

	_, db := heavy.CreateDBConnection()

	dataRequestor := processing.NewDataRequestor(db, rpcBus, responseChan)

	reader := &Reader{
		Connection: pconn,
		exitChan:   exitChan,
		router: &messageRouter{
			publisher:       publisher,
			dupeMap:         dupeMap,
			blockHashBroker: processing.NewBlockHashBroker(db, responseChan),
			synchronizer:    chainsync.NewChainSynchronizer(publisher, rpcBus, responseChan, counter),
			dataRequestor:   dataRequestor,
			dataBroker:      processing.NewDataBroker(db, rpcBus, responseChan),
			ponger:          processing.NewPonger(responseChan),
			peerInfo:        conn.RemoteAddr().String(),
		},
	}

	// On each new connection the node sends topics.Mempool to retrieve mempool
	// txs from the new peer
	go func() {
		if err := dataRequestor.RequestMempoolItems(); err != nil {
			log.WithFields(log.Fields{
				"process": "peer",
				"error":   err,
			}).Warnln("error sending topics.Mempool message")
		}
	}()

	return reader, nil
}

// ReadMessage reads from the connection
func (c *Connection) ReadMessage() ([]byte, error) {
	// COBS  c.reader.ReadBytes(0x00)
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
func (p *Writer) Connect() error {
	if err := p.Handshake(); err != nil {
		p.Conn.Close()
		return err
	}

	return nil
}

// Accept will perform the protocol handshake with the peer.
func (p *Reader) Accept() error {
	if err := p.Handshake(); err != nil {
		p.Conn.Close()
		return err
	}

	return nil
}

// Serve utilizes two different methods for writing to the open connection
func (w *Writer) Serve(writeQueueChan <-chan *bytes.Buffer, exitChan chan struct{}) {

	defer w.onDisconnect()

	// Any gossip topics are written into interrupt-driven ringBuffer
	// Single-consumer pushes messages to the socket
	w.gossipID = w.subscriber.Subscribe(topics.Gossip, eventbus.NewStreamListener(w.Connection))

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
			log.WithFields(log.Fields{
				"process": "peer",
				"error":   err,
			}).Warnln("error pinging peer")
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

	if err := w.Connection.gossip.Process(buf); err != nil {
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
				log.WithFields(log.Fields{
					"process": "peer",
					"error":   err,
				}).Warnln("error processing outgoing message")
				continue
			}

			if _, err := w.Connection.Write(buf.Bytes()); err != nil {
				log.WithFields(log.Fields{
					"process": "peer",
					"queue":   "writequeue",
					"error":   err,
				}).Warnln("error writing message")
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
	defer p.Conn.Close()
	defer func() {
		p.exitChan <- struct{}{}
	}()

	for {
		// Refresh the read deadline
		p.Conn.SetReadDeadline(time.Now().Add(readWriteTimeout))

		b, err := p.ReadMessage()
		if err != nil {
			log.WithFields(log.Fields{
				"process": "peer",
				"error":   err,
			}).Warnln("error reading message")
			return
		}

		p.router.Collect(bytes.NewBuffer(b))
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
