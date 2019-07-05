package peer

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
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
	reader *bufio.Reader
	magic  protocol.Magic
}

// Writer abstracts all of the logic and fields needed to write messages to
// other network nodes.
type Writer struct {
	*Connection
	gossip   *processing.Gossip
	gossipID uint32
	// TODO: add service flag
}

// Reader abstracts all of the logic and fields needed to receive messages from
// other network nodes.
type Reader struct {
	*Connection
	unmarshaller *messageUnmarshaller
	router       *messageRouter
	// TODO: add service flag
}

// NewWriter returns a Writer. It will still need to be initialized by
// subscribing to the gossip topic with a stream handler, and by running the WriteLoop
// in a goroutine..
func NewWriter(conn net.Conn, magic protocol.Magic, subscriber wire.EventSubscriber) *Writer {
	pw := &Writer{
		Connection: &Connection{
			Conn:   conn,
			reader: bufio.NewReader(conn),
			magic:  magic,
		},
		gossip: processing.NewGossip(magic),
	}

	subscriber.RegisterPreprocessor(string(topics.Gossip), pw.gossip)
	return pw
}

// NewReader returns a Reader. It will still need to be initialized by
// running ReadLoop in a goroutine.
func NewReader(conn net.Conn, magic protocol.Magic, dupeMap *dupemap.DupeMap, publisher wire.EventPublisher,
	rpcBus *wire.RPCBus, counter *chainsync.Counter, responseChan chan<- *bytes.Buffer) (*Reader, error) {
	pconn := &Connection{
		Conn:   conn,
		reader: bufio.NewReader(conn),
		magic:  magic,
	}

	db, err := OpenDB()
	if err != nil {
		return nil, err
	}

	dataRequestor := processing.NewDataRequestor(db, rpcBus, responseChan)

	reader := &Reader{
		Connection:   pconn,
		unmarshaller: &messageUnmarshaller{magic},
		router: &messageRouter{
			publisher:       publisher,
			dupeMap:         dupeMap,
			blockHashBroker: processing.NewBlockHashBroker(db, responseChan),
			synchronizer:    chainsync.NewChainSynchronizer(publisher, rpcBus, responseChan, counter),
			dataRequestor:   dataRequestor,
			dataBroker:      processing.NewDataBroker(db, rpcBus, responseChan),
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

// ReadMessage reads from the connection until encountering a zero byte.
func (c *Connection) ReadMessage() ([]byte, error) {
	return c.reader.ReadBytes(0x00)
}

// Connect will perform the protocol handshake with the peer. If successful
func (p *Writer) Connect(subscriber wire.EventSubscriber) error {
	if err := p.Handshake(); err != nil {
		p.Conn.Close()
		return err
	}

	p.Subscribe(subscriber)
	return nil
}

// Subscribe the writer to the gossip topic, passing it's connection as the writer.
func (p *Writer) Subscribe(subscriber wire.EventSubscriber) {
	id := subscriber.SubscribeStream(string(topics.Gossip), p.Connection)
	p.gossipID = id
}

// Accept will perform the protocol handshake with the peer.
func (p *Reader) Accept() error {
	if err := p.Handshake(); err != nil {
		p.Conn.Close()
		return err
	}

	return nil
}

// WriteLoop waits for messages to arrive on the `responseChan`, which are intended
// only for this specific peer. The messages get prepended with a magic, and then
// are COBS encoded before being sent over the wire.
func (w *Writer) WriteLoop(responseChan <-chan *bytes.Buffer) {
	defer w.Conn.Close()

	for {
		buf := <-responseChan
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
				"error":   err,
			}).Warnln("error writing message")
			return
		}
	}
}

// ReadLoop will block on the read until a message is read, or until the deadline
// is reached. Should be called in a go-routine, after a successful handshake with
// a peer. Eventual duplicated messages are silently discarded.
func (p *Reader) ReadLoop() {
	defer p.Conn.Close()

	for {
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

		// Refresh the read deadline
		p.Conn.SetReadDeadline(time.Now().Add(readWriteTimeout))
	}
}

// Read the topic bytes off r, and return them as a topics.Topic.
func extractTopic(r io.Reader) topics.Topic {
	var cmdBuf [topics.Size]byte
	if _, err := r.Read(cmdBuf[:]); err != nil {
		panic(err)
	}
	return topics.ByteArrayToTopic(cmdBuf)
}

// Read the magic bytes off r, and return them as a protocol.Magic.
func extractMagic(r io.Reader) protocol.Magic {
	buffer := make([]byte, 4)
	if _, err := r.Read(buffer); err != nil {
		panic(err)
	}

	magic := binary.LittleEndian.Uint32(buffer)
	return protocol.Magic(magic)
}

// Write a message to the connection.
// Conn needs to be locked, as this function can be called both by the WriteLoop,
// and by the writer on the ring buffer.
func (c *Connection) Write(b []byte) (int, error) {
	c.Conn.SetWriteDeadline(time.Now().Add(readWriteTimeout))
	c.lock.Lock()
	n, err := c.Conn.Write(b)
	c.lock.Unlock()
	return n, err
}

// Addr returns the peer's address as a string.
func (c *Connection) Addr() string {
	return c.Conn.RemoteAddr().String()
}

func OpenDB() (database.DB, error) {
	drvr, err := database.From(cfg.Get().Database.Driver)
	if err != nil {
		return nil, err
	}

	db, err := drvr.Open(cfg.Get().Database.Dir, protocol.MagicFromConfig(), true)
	if err != nil {
		return nil, err
	}

	return db, nil
}
