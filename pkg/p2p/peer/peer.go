package peer

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/heavy"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/dupemap"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing/chainsync"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

var readWriteTimeout = 60 * time.Second // Max idle time for a peer

// Connection holds the TCP connection to another node, and it's known protocol magic.
// The `net.Conn` is guarded by a mutex, to allow both multicast and one-to-one
// communication between peers.
type Connection struct {
	lock sync.Mutex
	net.Conn
	magic protocol.Magic
}

// Writer abstracts all of the logic and fields needed to write messages to
// other network nodes.
type Writer struct {
	*Connection
	gossip     *processing.Gossip
	gossipID   uint32
	subscriber wire.EventSubscriber
	// TODO: add service flag
}

// Reader abstracts all of the logic and fields needed to receive messages from
// other network nodes.
type Reader struct {
	*Connection
	unmarshaller *messageUnmarshaller
	router       *messageRouter
	exitChan     chan<- struct{} // Way to kill the WriteLoop
	// TODO: add service flag
}

// NewWriter returns a Writer. It will still need to be initialized by
// subscribing to the gossip topic with a stream handler, and by running the WriteLoop
// in a goroutine..
func NewWriter(conn net.Conn, magic protocol.Magic, subscriber wire.EventSubscriber) *Writer {
	pw := &Writer{
		Connection: &Connection{
			Conn:  conn,
			magic: magic,
		},
		gossip:     processing.NewGossip(magic),
		subscriber: subscriber,
	}

	return pw
}

// NewReader returns a Reader. It will still need to be initialized by
// running ReadLoop in a goroutine.
func NewReader(conn net.Conn, magic protocol.Magic, dupeMap *dupemap.DupeMap, publisher wire.EventPublisher, rpcBus *wire.RPCBus, counter *chainsync.Counter, responseChan chan<- *bytes.Buffer, exitChan chan<- struct{}) (*Reader, error) {
	pconn := &Connection{
		Conn:  conn,
		magic: magic,
	}

	_, db := heavy.CreateDBConnection()

	dataRequestor := processing.NewDataRequestor(db, rpcBus, responseChan)

	reader := &Reader{
		Connection:   pconn,
		unmarshaller: &messageUnmarshaller{magic},
		exitChan:     exitChan,
		router: &messageRouter{
			publisher:       publisher,
			dupeMap:         dupeMap,
			blockHashBroker: processing.NewBlockHashBroker(db, responseChan),
			synchronizer:    chainsync.NewChainSynchronizer(publisher, rpcBus, responseChan, counter),
			dataRequestor:   dataRequestor,
			dataBroker:      processing.NewDataBroker(db, rpcBus, responseChan),
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
	return processing.ReadFrame(c.Conn)
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
	w.gossipID = w.subscriber.SubscribeStream(string(topics.Gossip), w.Connection)

	// writeQueue - FIFO queue
	// writeLoop pushes first-in message to the socket
	w.writeLoop(writeQueueChan, exitChan)
}

func (w *Writer) onDisconnect() {
	log.Infof("Connection to %s terminated", w.Connection.RemoteAddr().String())
	w.Conn.Close()
	w.subscriber.Unsubscribe(string(topics.Gossip), w.gossipID)
}

func (w *Writer) writeLoop(writeQueueChan <-chan *bytes.Buffer, exitChan chan struct{}) {

	for {
		select {
		case buf := <-writeQueueChan:
			processed, err := w.gossip.Process(buf)
			if err != nil {
				log.WithFields(log.Fields{
					"process": "peer",
					"error":   err,
				}).Warnln("error processing outgoing message")
				continue
			}

			if _, err := w.Connection.Write(processed.Bytes()); err != nil {
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

		buf := new(bytes.Buffer)
		if err := p.unmarshaller.Unmarshal(b, buf); err != nil {
			log.WithFields(log.Fields{
				"process": "peer",
				"error":   err,
			}).Warnln("error unmarshalling message")
			continue
		}

		p.router.Collect(buf)
	}
}

// Read the topic bytes off r, and return them as a topics.Topic.
func extractTopic(r io.Reader) (topics.Topic, error) {
	var cmdBuf [topics.Size]byte
	if _, err := r.Read(cmdBuf[:]); err != nil {
		return topics.Topic(""), err
	}
	return topics.ByteArrayToTopic(cmdBuf), nil
}

// Read the magic bytes off r, and return them as a protocol.Magic.
func extractMagic(r io.Reader) (protocol.Magic, error) {
	buffer := make([]byte, 4)
	if _, err := r.Read(buffer); err != nil {
		return protocol.Magic(0), err
	}

	magic := binary.LittleEndian.Uint32(buffer)
	return protocol.Magic(magic), nil
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
