package peer

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/dupemap"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

var readWriteTimeout = 2 * time.Minute // Used to set reading and writing deadlines

// Connection holds the TCP connection to another node, and it's known protocol magic.
type Connection struct {
	lock sync.Mutex
	net.Conn
	magic protocol.Magic
}

// Writer abstracts all of the logic and fields needed to write messages to
// other network nodes.
type Writer struct {
	*Connection
	subscriber wire.EventSubscriber
	gossipID   uint32
	// TODO: add service flag
}

// Reader abstracts all of the logic and fields needed to receive messages from
// other network nodes.
type Reader struct {
	*Connection
	unmarshaller *messageUnmarshaller
	// TODO: add service flag
}

// MessageCollector is responsible for handling decoded messages from the wire.
type MessageCollector struct {
	Publisher     wire.EventPublisher
	DupeBlacklist *dupemap.DupeMap
	Magic         protocol.Magic
}

// Collect a decoded message from the Reader.
func (m *MessageCollector) Collect(b *bytes.Buffer) error {
	// check if this message is a duplicate of another we already forwarded
	if m.DupeBlacklist.CanFwd(b) {
		topic := extractTopic(b)
		m.Publisher.Publish(string(topic), b)
	}

	return nil
}

// NewWriter returns a Writer. It will still need to be initialized by
// subscribing to the gossip topic with a stream handler.
func NewWriter(conn net.Conn, magic protocol.Magic, subscriber wire.EventSubscriber) *Writer {
	pw := &Writer{
		Connection: &Connection{
			Conn:  conn,
			magic: magic,
		},
		subscriber: subscriber,
	}

	gossip := NewGossip(magic)
	subscriber.RegisterPreprocessor(string(topics.Gossip), gossip)
	return pw
}

// NewReader returns a Reader. It will still need to be initialized by
// running ReadLoop in a goroutine.
func NewReader(conn net.Conn, magic protocol.Magic) *Reader {
	return &Reader{
		Connection: &Connection{
			Conn:  conn,
			magic: magic,
		},
		unmarshaller: &messageUnmarshaller{magic},
	}
}

// ReadMessage reads from the connection until encountering a zero byte.
func (c *Connection) ReadMessage() ([]byte, error) {
	r := bufio.NewReader(c.Conn)
	return r.ReadBytes(0x00)
}

// Connect will perform the protocol handshake with the peer. If successful
func (p *Writer) Connect() error {
	if err := p.Handshake(); err != nil {
		p.Conn.Close()
		return err
	}

	p.Subscribe()
	return nil
}

// Subscribe the writer to the gossip topic, passing it's connection as the writer.
func (p *Writer) Subscribe() {
	id := p.subscriber.SubscribeStream(string(topics.Gossip), p.Connection)
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

// ReadLoop will block on the read until a message is read, or until the deadline
// is reached. Should be called in a go-routine, after a successful handshake with
// a peer. Eventual duplicated messages are silently discarded.
func (p *Reader) ReadLoop(c wire.EventCollector) {
	defer func() {
		p.Conn.Close()
	}()

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

		c.Collect(buf)

		// Refresh the read deadline
		p.Conn.SetReadDeadline(time.Now().Add(readWriteTimeout))
	}
}

func extractTopic(r io.Reader) topics.Topic {
	var cmdBuf [topics.Size]byte
	if _, err := r.Read(cmdBuf[:]); err != nil {
		panic(err)
	}

	return topics.ByteArrayToTopic(cmdBuf)
}

func extractMagic(r io.Reader) protocol.Magic {
	buffer := make([]byte, 4)
	if _, err := r.Read(buffer); err != nil {
		panic(err)
	}

	magic := binary.LittleEndian.Uint32(buffer)
	return protocol.Magic(magic)
}

// Write a message to the connection.
func (c *Connection) Write(b []byte) (int, error) {
	c.lock.Lock()
	n, err := c.Conn.Write(b)
	c.lock.Unlock()
	return n, err
}

// Port returns the port
func (c *Connection) Port() uint16 {
	s := strings.Split(c.Addr(), ":")
	port, _ := strconv.ParseUint(s[1], 10, 16)
	return uint16(port)
}

// Addr returns the peer's address as a string.
func (c *Connection) Addr() string {
	return c.Conn.RemoteAddr().String()
}
