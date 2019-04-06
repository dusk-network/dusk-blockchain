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
	"sync/atomic"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/chain"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/stall"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

const (
	maxOutboundConnections = 100
	handshakeTimeout       = 30 * time.Second
	idleTimeout            = 2 * time.Minute // If no message received after idleTimeout, then peer disconnects

	// nodes will have `responseTime` seconds to reply with a response
	responseTime = 300 * time.Second

	// the stall detector will check every `tickerInterval` to see if messages
	// are overdue. Should be less than `responseTime`
	tickerInterval = 30 * time.Second

	// The input buffer size is the amount of mesages that
	// can be buffered into the channel to receive at once before
	// blocking, and before determinism is broken
	inputBufferSize = 100

	// The output buffer size is the amount of messages that
	// can be buffered into the channel to send at once before
	// blocking, and before determinism is broken.
	outputBufferSize = 100

	// pingInterval = 20 * time.Second //Not implemented in Dusk clients
	obsoleteMessageRound = 3
)

var (
	errHandShakeTimeout    = errors.New("Handshake timed out, peers have " + handshakeTimeout.String() + " to complete the handshake")
	errHandShakeFromStr    = "Handshake failed: %s"
	receivedMessageFromStr = "Received a '%s' message from %s"
)

// Config holds a set of functions needed to directly work with the blockchain,
// in case of a individual request (synchronization)
type Config struct {
	GetHeaders func([]byte, []byte) []*block.Header
	GetBlock   func([]byte) *block.Block
}

// Peer holds all configuration and state to be able to communicate with other peers.
// Every Peer has a Detector that keeps track of pending messages that require a synchronous response.
// The peer also holds a pointer to the chain, to directly request certain information.
type Peer struct {
	// Unchangeable state: concurrent safe
	Addr      string
	ProtoVer  *protocol.Version
	Inbound   bool
	Services  protocol.ServiceFlag
	CreatedAt time.Time
	Relay     bool

	Conn     net.Conn
	eventBus *wire.EventBus
	chain    *chain.Chain
	magic    protocol.Magic

	// Atomic vals
	Disconnected int32
	syncing      int32

	VerackReceived bool
	VersionKnown   bool

	*stall.Detector

	inch   chan func() // Will handle all inbound connections from peer
	outch  chan func() // Will handle all outbound connections to peer
	quitch chan struct{}

	dupeBlacklist *dupeMap
}

type dupeMap struct {
	roundChan <-chan uint64 // Will get notification about new rounds in order...
	tmpMap    *TmpMap       // ...to clean up the duplicate message blacklist
}

func newDupeMap(eventbus *wire.EventBus) *dupeMap {
	roundChan := consensus.InitRoundUpdate(eventbus)
	tmpMap := NewTmpMap(obsoleteMessageRound)
	return &dupeMap{
		roundChan,
		tmpMap,
	}
}

func (d *dupeMap) cleanOnRound() {
	for {
		round := <-d.roundChan
		d.tmpMap.UpdateHeight(round)
	}
}

func (d *dupeMap) canFwd(payload *bytes.Buffer) bool {
	found := d.tmpMap.HasAnywhere(payload)
	if found {
		return false
	}
	d.tmpMap.Add(payload)
	return true
}

// NewPeer is called after a connection to a peer was successful.
// Inbound as well as Outbound.
func NewPeer(conn net.Conn, inbound bool, magic protocol.Magic, eventBus *wire.EventBus) *Peer {

	dupeBlacklist := newDupeMap(eventBus)
	go dupeBlacklist.cleanOnRound()

	p := &Peer{
		inch:          make(chan func(), inputBufferSize),
		outch:         make(chan func(), outputBufferSize),
		quitch:        make(chan struct{}, 1),
		Inbound:       inbound,
		Conn:          conn,
		eventBus:      eventBus,
		Addr:          conn.RemoteAddr().String(),
		Detector:      stall.NewDetector(responseTime, tickerInterval),
		magic:         magic,
		dupeBlacklist: dupeBlacklist,
	}

	return p
}

// Collect implements wire.EventCollector.
// Receive messages, add headers, and propagate them to the network.
func (p *Peer) Collect(message *bytes.Buffer) error {
	// copy the buffer here, as the pointer we just got is shared by
	// all the other peers.
	msg := *message
	var topicBytes [15]byte

	_, _ = msg.Read(topicBytes[:])
	topic := topics.ByteArrayToTopic(topicBytes)
	return p.WriteMessage(&msg, topic)
}

// WriteMessage will append a header with the specified topic to the passed
// message, and write it to the peer.
func (p *Peer) WriteMessage(msg *bytes.Buffer, topic topics.Topic) error {
	messageWithHeader, err := addHeader(msg, p.magic, topic)
	if err != nil {
		return err
	}

	p.Write(messageWithHeader)
	return nil
}

// Write to a peer
func (p *Peer) Write(msg *bytes.Buffer) {
	p.outch <- func() {
		_, _ = p.Conn.Write(msg.Bytes())
	}
}

// Disconnect disconnects from a peer
func (p *Peer) Disconnect() {
	// return if already disconnected
	if atomic.LoadInt32(&p.Disconnected) != 0 {
		return
	}

	atomic.AddInt32(&p.Disconnected, 1)

	p.Detector.Quit()
	close(p.quitch)
	p.Conn.Close()
}

// Port returns the port
func (p *Peer) Port() uint16 {
	s := strings.Split(p.Conn.RemoteAddr().String(), ":")
	port, _ := strconv.ParseUint(s[1], 10, 16)
	return uint16(port)
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
	if _, err := io.ReadFull(p.Conn, buffer); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (p *Peer) readPayload(length uint32) (*bytes.Buffer, error) {
	buffer := make([]byte, length)
	if _, err := io.ReadFull(p.Conn, buffer); err != nil {
		return nil, err
	}

	return bytes.NewBuffer(buffer), nil
}

func (p *Peer) headerMagicIsValid(header *MessageHeader) bool {
	return p.magic == header.Magic
}

func payloadChecksumIsValid(payloadBuffer *bytes.Buffer, checksum uint32) bool {
	return crypto.CompareChecksum(payloadBuffer.Bytes(), checksum)
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

	if !payloadChecksumIsValid(payloadBuffer, header.Checksum) {
		return "", nil, errors.New("invalid payload checksum")
	}

	return header.Topic, payloadBuffer, nil
}

// Run is used to start communicating with the peer, completes the handshake and starts observing
// for messages coming in and allows for queing of outgoing messages
func (p *Peer) Run() error {
	go p.writeLoop()
	if err := p.Handshake(); err != nil {
		p.Disconnect()
		return err
	}

	go wire.NewEventSubscriber(p.eventBus, p, string(topics.Gossip)).Accept()
	go p.startProtocol()
	go p.readLoop()
	return nil
}

// StartProtocol is run as a go-routine, will act as our queue for messages.
// Should be ran after handshake
func (p *Peer) startProtocol() {
loop:
	for atomic.LoadInt32(&p.Disconnected) == 0 {
		select {
		case <-p.quitch:
			break loop
		case <-p.Detector.Quitch:
			break loop
		}
	}
	p.Disconnect()
}

// ReadLoop will block on the read until a message is read.
// Should only be called after handshake is complete on a seperate go-routine.
// Eventual duplicated messages are silently discarded
func (p *Peer) readLoop() {
	idleTimer := time.AfterFunc(idleTimeout, func() {
		p.Disconnect()
	})

	for atomic.LoadInt32(&p.Disconnected) == 0 {
		idleTimer.Reset(idleTimeout) // reset timer on each loop
		topic, payload, err := p.readMessage()
		if err != nil {
			p.Disconnect()
			return
		}

		// check if this message is a duplicate of another we already forwarded
		if p.dupeBlacklist.canFwd(payload) {
			p.eventBus.Publish(string(topic), payload)
		}
	}
}

// WriteLoop will queue all messages to be written to the peer
func (p *Peer) writeLoop() {
	for atomic.LoadInt32(&p.Disconnected) == 0 {
		f := <-p.outch
		f()
	}
}
