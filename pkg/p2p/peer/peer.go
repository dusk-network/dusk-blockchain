// Package peer uses channels to simulate the queue handler with the actor model.
// A suitable number k ,should be set for channel size, because if #numOfMsg > k, we lose determinism.
// k chosen should be large enough that when filled, it shall indicate that the peer has stopped
// responding, since we do not have a pingMSG, we will need another way to shut down peers.
package peer

import (
	"bytes"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/dupemap"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

const (
	handshakeTimeout = 30 * time.Second // Used to set handshake deadline
	readWriteTimeout = 2 * time.Minute  // Used to set reading and writing deadlines
)

// Peer holds all configuration and state to be able to communicate with other peers.
type Peer struct {
	conn  net.Conn
	magic protocol.Magic
	// Services protocol.ServiceFlag - currently not implemented

	eventBus     *wire.EventBus
	quitID       uint32
	gossipID     uint32
	outgoingChan chan func() error // outgoing message queue

	dupeBlacklist *dupemap.DupeMap
}

// NewPeer is called after a connection to a peer was successful.
// Inbound as well as Outbound.
func NewPeer(conn net.Conn, magic protocol.Magic, eventBus *wire.EventBus,
	dupeMap *dupemap.DupeMap) *Peer {
	return &Peer{
		outgoingChan:  make(chan func() error, 100),
		conn:          conn,
		eventBus:      eventBus,
		magic:         magic,
		dupeBlacklist: dupeMap,
	}
}

// Connect will perform the protocol handshake with the peer. If successful, we start
// a set of goroutines which are used for communication.
func (p *Peer) Connect(inbound bool) error {
	if err := p.Handshake(inbound); err != nil {
		p.Disconnect()
		return err
	}

	go p.writeLoop()
	go p.readLoop()
	p.subscribeToGossipEvents()
	return nil
}

// ReadLoop will block on the read until a message is read, or until the deadline
// is reached. Should be called in a go-routine, after a successful handshake with
// a peer. Eventual duplicated messages are silently discarded.
func (p *Peer) readLoop() {
	for {
		// Refresh the read deadline
		p.conn.SetReadDeadline(time.Now().Add(readWriteTimeout))

		topic, payload, err := p.readMessage()
		if err != nil {
			log.WithFields(log.Fields{
				"process": "peer",
				"error":   err,
			}).Warnln("error reading message")
			p.Disconnect()
			return
		}

		// check if this message is a duplicate of another we already forwarded
		if p.dupeBlacklist.CanFwd(payload) {
			p.eventBus.Publish(string(topic), payload)
		}
	}
}

// WriteLoop will write queued messages to the peer in a synchronous fashion.
// Should be called in a go-routine, after a successful handshake with a peer.
func (p *Peer) writeLoop() {
	for {
		f := <-p.outgoingChan

		// Refresh the write deadline
		p.conn.SetWriteDeadline(time.Now().Add(readWriteTimeout))

		if err := f(); err != nil {
			log.WithFields(log.Fields{
				"process": "peer",
				"error":   err,
			}).Warnln("error writing message")
			p.Disconnect()
			return
		}
	}
}

// Collect implements wire.EventCollector.
// Receive messages from other components, add headers, and propagate them to the network.
func (p *Peer) Collect(message *bytes.Buffer) error {
	// copy the buffer here, as the pointer we just got is shared by
	// all the other peers.
	msg := *message
	var topicBytes [15]byte

	_, _ = msg.Read(topicBytes[:])
	topic := topics.ByteArrayToTopic(topicBytes)
	return p.writeMessage(&msg, topic)
}

// WriteMessage will append a header with the specified topic to the passed
// message, and put it in the outgoing message queue.
func (p *Peer) writeMessage(msg *bytes.Buffer, topic topics.Topic) error {
	messageWithHeader, err := addHeader(msg, p.magic, topic)
	if err != nil {
		return err
	}

	p.write(messageWithHeader)
	return nil
}

// Write will put a message in the outgoing message queue.
func (p *Peer) write(msg *bytes.Buffer) {
	p.outgoingChan <- func() error {
		_, err := p.conn.Write(msg.Bytes())
		return err
	}
}

// Disconnect disconnects from a peer
func (p *Peer) Disconnect() {
	log.WithFields(log.Fields{
		"process": "peer",
		"address": p.conn.RemoteAddr().String(),
	}).Warnln("peer disconnected")

	_ = p.conn.Close()
	if p.gossipID != 0 && p.quitID != 0 {
		_ = p.eventBus.Unsubscribe(string(topics.Gossip), p.gossipID)
		_ = p.eventBus.Unsubscribe(wire.QuitTopic, p.quitID)
	}
}

// Port returns the port
func (p *Peer) Port() uint16 {
	s := strings.Split(p.conn.RemoteAddr().String(), ":")
	port, _ := strconv.ParseUint(s[1], 10, 16)
	return uint16(port)
}

// Addr returns the peer's address as a string.
func (p *Peer) Addr() string {
	return p.conn.RemoteAddr().String()
}

// Read from a peer
func (p *Peer) readHeader() (*MessageHeader, error) {
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

func (p *Peer) readHeaderBytes() ([]byte, error) {
	buffer := make([]byte, MessageHeaderSize)
	if _, err := io.ReadFull(p.conn, buffer); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (p *Peer) readPayload(length uint32) (*bytes.Buffer, error) {
	buffer := make([]byte, length)
	if _, err := io.ReadFull(p.conn, buffer); err != nil {
		return nil, err
	}

	return bytes.NewBuffer(buffer), nil
}

func (p *Peer) headerMagicIsValid(header *MessageHeader) bool {
	return p.magic == header.Magic
}

func (p *Peer) readMessage() (topics.Topic, *bytes.Buffer, error) {
	header, err := p.readHeader()
	if err != nil {
		return "", nil, err
	}

	if !p.headerMagicIsValid(header) {
		return "", nil, errors.New("invalid header magic")
	}

	payloadBuffer, err := p.readPayload(header.Length)
	if err != nil {
		return "", nil, err
	}

	return header.Topic, payloadBuffer, nil
}

func (p *Peer) subscribeToGossipEvents() {
	es := wire.NewTopicListener(p.eventBus, p, string(topics.Gossip))
	p.quitID = es.QuitChanID
	p.gossipID = es.MsgChanID
	go es.Accept()
}
