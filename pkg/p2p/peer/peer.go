package peer

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/dupemap"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

var readWriteTimeout = 2 * time.Minute // Used to set reading and writing deadlines

var zeroSplit = func(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if i := bytes.IndexByte(data, 0x00); i >= 0 {
		return i + 1, data, nil
	}
	if atEOF {
		return 0, nil, io.EOF
	}
	return 0, nil, nil
}

// Connection holds the TCP connection to another node, and it's known protocol magic.
type Connection struct {
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

// Connect will perform the protocol handshake with the peer. If successful
func (p *Writer) Connect() error {
	if err := p.HandShake(); err != nil {
		p.Conn.Close()
		return err
	}

	p.Subscribe()
	p.Conn.SetWriteDeadline(time.Time{})
	return nil
}

// Subscribe the writer to the gossip topic, passing it's connection as the writer.
func (p *Writer) Subscribe() {
	id := p.subscriber.SubscribeStream(string(topics.Gossip), p.Conn)
	p.gossipID = id
}

// Accept will perform the protocol handshake with the peer.
func (p *Reader) Accept() error {
	if err := p.HandShake(); err != nil {
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

	r := bufio.NewReader(p.Conn)
	for {
		b, err := r.ReadBytes(0x00)
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

func extractTopic(p io.Reader) topics.Topic {
	var cmdBuf [topics.Size]byte
	if _, err := p.Read(cmdBuf[:]); err != nil {
		panic(err)
	}

	return topics.ByteArrayToTopic(cmdBuf)
}

// Write will put a message in the outgoing message queue.
func (p *Writer) Write(b []byte) (int, error) {
	return p.Conn.Write(b)
}

// Port returns the port
func (p *Connection) Port() uint16 {
	s := strings.Split(p.Conn.RemoteAddr().String(), ":")
	port, _ := strconv.ParseUint(s[1], 10, 16)
	return uint16(port)
}

// Addr returns the peer's address as a string.
func (p *Connection) Addr() string {
	return p.Conn.RemoteAddr().String()
}

// Read from a peer
func (p *Connection) readHeader() (*MessageHeader, error) {
	headerBytes, err := p.readHeaderBytes()
	if err != nil {
		return nil, err
	}

	headerBuffer := bytes.NewReader(headerBytes)
	header, err := decodeMessageHeader(headerBuffer)
	if err != nil {
		return nil, err
	}

	return header, nil
}

func (p *Connection) readHeaderBytes() ([]byte, error) {
	buffer := make([]byte, MessageHeaderSize)
	if _, err := io.ReadFull(p.Conn, buffer); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (m *MessageCollector) magicIsValid(b *bytes.Buffer) bool {
	var magic uint32
	if err := encoding.ReadUint32(b, binary.LittleEndian, &magic); err != nil {
		panic(err)
	}

	return m.Magic == protocol.Magic(magic)
}
