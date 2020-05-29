package kadcast

import (
	"bytes"
	"errors"
	"net"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

// Reader is running a TCP listner handling TCP Broadcast packets from kadcast network
//
// On a TCP connection
// Read a single tcp packet from the accepted connection
// Validate the packet
// Extract Gossip Packet
// Build and Publish eventBus message
type Reader struct {
	publisher eventbus.Publisher
	gossip    *protocol.Gossip

	// lpeer is the tuple identifying this peer
	lpeer encoding.PeerInfo

	listener      *net.TCPListener
	messageRouter messageRouter
}

// NewReader makes a new kadcast reader that handles TCP packets of broadcasting
func NewReader(lpeerInfo encoding.PeerInfo, publisher eventbus.Publisher, gossip *protocol.Gossip, dupeMap *dupemap.DupeMap) *Reader {

	addr := lpeerInfo.Address()
	lAddr, err := net.ResolveTCPAddr("tcp4", addr)
	if err != nil {
		log.Panicf("invalid kadcast peer address %s", addr)
	}

	l, err := net.ListenTCP("tcp4", lAddr)
	if err != nil {
		log.Panic(err)
	}

	reader := &Reader{
		listener:      l,
		lpeer:         lpeerInfo,
		messageRouter: messageRouter{publisher: publisher, dupeMap: dupeMap},
		publisher:     publisher,
		gossip:        gossip,
	}

	log.WithField("l_addr", lAddr.String()).Infoln("Starting Reader")

	return reader
}

// Close closes reader TCP listener
func (r *Reader) Close() error {
	if r.listener != nil {
		return r.listener.Close()
	}
	return nil
}

// Serve starts accepting and processing TCP connection and packets
func (r *Reader) Serve() {

	for {
		conn, err := r.listener.AcceptTCP()
		if err != nil {
			log.WithError(err).Warn("Error on tcp accept")
			return
		}

		// processPacket as for now a single-packet-per-connection is allowed
		go r.processPacket(conn)
	}
}
func (r *Reader) processPacket(conn *net.TCPConn) {

	// As the peer readloop is at the front-line of P2P network, receiving a
	// malformed frame by an adversary node could lead to a panic.
	defer func() {
		if r := recover(); r != nil {
			log.Error("readloop failed: ", r)
		}
	}()

	// Current impl expects only one TCPFrame per connection
	defer func() {
		if conn != nil {
			_ = conn.Close()
		}
	}()

	remotePeerAddr := conn.RemoteAddr().String()
	llog := log.WithField("l_addr", r.lpeer.String()).
		WithField("r_addr", remotePeerAddr)

	// Read frame payload Set a new deadline for the connection.
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	b, err := readTCPFrame(conn)
	if err != nil {
		log.WithError(err).Warn("Error on frame read")
		return
	}

	// Unmarshal message header
	buf := bytes.NewBuffer(b)
	var header encoding.Header
	err = header.UnmarshalBinary(buf)
	if err != nil {
		llog.WithError(err).Warn("TCP reader rejects a packet")
		return
	}

	// Run extra checks over message data
	if err := isValidMessage(remotePeerAddr, header); err != nil {
		llog.WithError(err).Warn("TCP reader rejects a packet")
		return
	}

	// Unmarshal broadcast message payload
	var p encoding.BroadcastPayload
	if err := p.UnmarshalBinary(buf); err != nil {
		llog.WithError(err).Warn("could not unmarshal message")
	}

	// Handle broadcast message
	if err := r.handleBroadcast(p.Height, p.GossipFrame); err != nil {
		llog.WithError(err).Warn("could not handle message")
	} else {
		llog.Traceln("Received Broadcast message")
	}
}

func (r *Reader) handleBroadcast(height byte, gossipFrame []byte) error {

	// Read `message` from gossip frame
	buf := bytes.NewBuffer(gossipFrame)
	message, err := r.gossip.ReadFrame(buf)
	if err != nil {
		log.WithError(err).Warnln("could not read the gossip frame")
		return err
	}

	// Propagate message to the node router respectively eventbus
	// Non-routable and duplicated messages are not repropagated
	err = r.messageRouter.Collect(message, height)
	if err != nil {
		log.WithError(err).Errorln("error routing message")
		return err
	}

	// Repropagate message here

	// From spec:
	//	When a node receives a CHUNK, it repeats the process in a store-and-
	//	forward manner: it buffers the data, picks a random node from its
	//	buckets up to (but not including) height h, and forwards the CHUNK with
	//	a smaller value for h accordingly.

	// NB Currently, repropagate in kadcast is fully delegated to the receiving
	// component. That's needed because only the receiving component is capable
	// of verifying message fully. E.g Chain component can verifies a new block

	return nil
}

func isValidMessage(remotePeerIP string, header encoding.Header) error {

	// Reader handles only broadcast-type messages
	if header.MsgType != encoding.BroadcastMsg {
		return errors.New("message type not supported")
	}

	// Make remote peerInfo based on addr from IP datagram and RemotePeerPort
	// from header
	remotePeer, err := encoding.MakePeerFromIP(remotePeerIP, header.RemotePeerPort)
	if err != nil {
		return err
	}

	// Ensure the RemotePeerID from header is correct one
	// This together with Nonce-PoW aims at providing a bit of DDoS protection
	if !bytes.Equal(remotePeer.ID[:], header.RemotePeerID[:]) {
		return errors.New("invalid remote peer id")
	}

	return nil
}
