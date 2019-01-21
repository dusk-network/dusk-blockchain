// Package peermgr uses channels to simulate the queue handler with the actor model.
// A suitable number k ,should be set for channel size, because if #numOfMsg > k, we lose determinism.
// k chosen should be large enough that when filled, it shall indicate that the peer has stopped
// responding, since we do not have a pingMSG, we will need another way to shut down peers.
package peermgr

import (
	"errors"
	log "github.com/sirupsen/logrus"
	cfg "github.com/spf13/viper"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/stall"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

const (
	maxOutboundConnections = 100
	handshakeTimeout       = 30 * time.Second
	idleTimeout            = 5 * time.Minute // If no message received after idleTimeout, then peer disconnects

	// nodes will have `responseTime` seconds to reply with a response
	responseTime = 120 * time.Second

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
)

var (
	errHandShakeTimeout    = errors.New("Handshake timed out, peers have " + handshakeTimeout.String() + " to complete the handshake")
	errHandShakeFromStr    = "Handshake failed: %s"
	receivedMessageFromStr = "Received a '%s' message from %s"
)

// ResponseHandler holds all response handlers related to response meesages from an external peer
type ResponseHandler struct {
	OnHeaders        func(*Peer, *payload.MsgHeaders)
	OnNotFound       func(*Peer, *payload.MsgNotFound)
	OnGetData        func(*Peer, *payload.MsgGetData)
	OnTx             func(*Peer, *payload.MsgTx)
	OnGetHeaders     func(*Peer, *payload.MsgGetHeaders)
	OnAddr           func(*Peer, *payload.MsgAddr)
	OnGetAddr        func(*Peer, *payload.MsgGetAddr)
	OnGetBlocks      func(*Peer, *payload.MsgGetBlocks)
	OnBlock          func(*Peer, *payload.MsgBlock)
	OnBinary         func(*Peer, *payload.MsgBinary)
	OnCertificate    func(*Peer, *payload.MsgCertificate)
	OnCertificateReq func(*Peer, *payload.MsgCertificateReq)
	OnMemPool        func(*Peer, *payload.MsgMemPool)
	OnPing           func(*Peer, *payload.MsgPing)
	OnPong           func(*Peer, *payload.MsgPong)
	OnReduction      func(*Peer, *payload.MsgReduction)
	OnReject         func(*Peer, *payload.MsgReject)
	OnScore          func(*Peer, *payload.MsgScore)
	OnInv            func(*Peer, *payload.MsgInv)
}

// Peer holds all configuration and state to be able to communicate with other peers.
// Every Peer has a Detector that keeps track of pending messages that require a synchronous response.
type Peer struct {
	// Unchangeable state: concurrent safe
	addr      string
	protoVer  uint32
	inbound   bool
	userAgent string
	services  protocol.ServiceFlag
	createdAt time.Time
	relay     bool
	Nonce     uint64

	net  protocol.Magic
	conn net.Conn

	hndlr ResponseHandler

	// Atomic vals
	disconnected int32

	statemutex     sync.Mutex
	verackReceived bool
	versionKnown   bool

	*stall.Detector

	inch   chan func() // Will handle all inbound connections from peer
	outch  chan func() // Will handle all outbound connections to peer
	quitch chan struct{}
}

// NewPeer is called after a connection to a peer was successful.
// Inbound as well as Outbound.
func NewPeer(conn net.Conn, inbound bool, respHndlr ResponseHandler) *Peer {
	p := Peer{}
	p.Nonce = rand.Uint64()
	p.hndlr = respHndlr
	p.inch = make(chan func(), inputBufferSize)
	p.outch = make(chan func(), outputBufferSize)
	p.quitch = make(chan struct{}, 1)
	p.inbound = inbound
	p.addr = conn.RemoteAddr().String()

	// TODO: Shouldn't this be sent by remote peer
	p.net = protocol.Magic(uint32(cfg.GetInt("net.magic")))

	p.conn = conn
	p.createdAt = time.Now()

	p.Detector = stall.NewDetector(responseTime, tickerInterval)

	return &p
}

// Write to a peer
func (p *Peer) Write(msg wire.Payload) error {
	return wire.WriteMessage(p.conn, p.net, msg)
}

// Read from a peer
func (p *Peer) Read() (wire.Payload, error) {
	return wire.ReadMessage(p.conn, p.net)
}

// Disconnect disconnects from a peer
func (p *Peer) Disconnect() {

	// return if already disconnected
	if atomic.LoadInt32(&p.disconnected) != 0 {
		return
	}

	atomic.AddInt32(&p.disconnected, 1)

	p.Detector.Quit()
	close(p.quitch)
	p.conn.Close()

	log.WithField("prefix", "peer").Infof("Disconnected peer with address %s", p.addr)
}

// ProtocolVersion returns the protocol version
func (p *Peer) ProtocolVersion() uint32 {
	return p.protoVer
}

// Net returns the protocol magic
func (p *Peer) Net() protocol.Magic {
	return p.net
}

// Port returns the port
func (p *Peer) Port() uint16 {
	s := strings.Split(p.RemoteAddr().String(), ":")
	port, _ := strconv.ParseUint(s[1], 10, 16)
	return uint16(port)
}

// CreatedAt returns the created at time
func (p *Peer) CreatedAt() time.Time {
	return p.createdAt
}

// CanRelay returns if the peer can be relayed
func (p *Peer) CanRelay() bool {
	return p.relay
}

// LocalAddr returns the local address of the peer
func (p *Peer) LocalAddr() net.Addr {
	return p.conn.LocalAddr()
}

// RemoteAddr returns the remote address of the peer
func (p *Peer) RemoteAddr() net.Addr {
	return p.conn.RemoteAddr()
}

// Services returns the services of the peer
func (p *Peer) Services() protocol.ServiceFlag {
	return p.services
}

// Inbound returns if the peer is inbound
func (p *Peer) Inbound() bool {
	return p.inbound
}

// UserAgent returns the user agent of the peer
func (p *Peer) UserAgent() string {
	return p.userAgent
}

// IsVerackReceived returns if the peer returned the 'verack' msg
func (p *Peer) IsVerackReceived() bool {
	return p.verackReceived
}

// NotifyDisconnect returns if the peer has disconnected
func (p *Peer) NotifyDisconnect() bool {
	log.WithField("prefix", "peer").Info("Peer has not disconnected yet")
	<-p.quitch
	log.WithField("prefix", "peer").Info("Peer has just disconnected")
	return true
}

//End of Exposed API functions//

// PingLoop not implemented yet.
// Will cause this client to disconnect from all other implementations
func (p *Peer) PingLoop() { /*not implemented in other neo clients*/ }

// Run is used to start communicating with the peer, completes the handshake and starts observing
// for messages coming in
func (p *Peer) Run() error {

	err := p.Handshake()

	go p.StartProtocol()
	go p.ReadLoop()
	go p.WriteLoop()

	//go p.PingLoop() // since it is not implemented. It will disconnect all other impls.
	return err

}

// StartProtocol is run as a go-routine, will act as our queue for messages.
// Should be ran after handshake
func (p *Peer) StartProtocol() {
loop:
	for atomic.LoadInt32(&p.disconnected) == 0 {
		select {
		case f := <-p.inch:
			f()
		case <-p.quitch:
			break loop
		case <-p.Detector.Quitch:
			log.WithField("prefix", "peer").Infof("Peer %s stalled, disconnecting", p.addr)
			break loop
		}
	}
	p.Disconnect()
}

// ReadLoop will block on the read until a message is read.
// Should only be called after handshake is complete on a seperate go-routine.
func (p *Peer) ReadLoop() {

	idleTimer := time.AfterFunc(idleTimeout, func() {
		log.WithField("prefix", "peer").Infof("Timing out peer %s", p.addr)
		p.Disconnect()
	})

loop:
	for atomic.LoadInt32(&p.disconnected) == 0 {

		idleTimer.Reset(idleTimeout) // reset timer on each loop

		readmsg, err := p.Read()

		// Message read; stop Timer
		idleTimer.Stop()

		if err != nil {
			// This will also happen if Peer is disconnected
			log.WithField("prefix", "peer").Infof("Connection to %s has been closed", p.addr)
			break loop
		}

		// Remove message as pending from the stall detector
		p.Detector.RemoveMessage(readmsg.Command())

		switch msg := readmsg.(type) {

		case *payload.MsgVersion:
			log.WithField("prefix", "peer").Infof("Already received a '%s' from %s, disconnecting", commands.Version, p.addr)
			break loop // We have already done the handshake, break loop and disconnect
		case *payload.MsgVerAck:
			if p.verackReceived {
				log.WithField("prefix", "peer").Infof("Already received a '%s' from %s , disconnecting", commands.VerAck, p.addr)
				break loop
			}
			p.statemutex.Lock() // This should not happen, however if it does, then we should set it.
			p.verackReceived = true
			p.statemutex.Unlock()
		case *payload.MsgAddr:
			p.OnAddr(msg)
		case *payload.MsgGetAddr:
			p.OnGetAddr(msg)
		case *payload.MsgGetBlocks:
			p.OnGetBlocks(msg)
		case *payload.MsgBlock:
			p.OnBlock(msg)
		case *payload.MsgHeaders:
			p.OnHeaders(msg)
		case *payload.MsgGetHeaders:
			p.OnGetHeaders(msg)
		case *payload.MsgInv:
			p.OnInv(msg)
		case *payload.MsgGetData:
			p.OnGetData(msg)
		case *payload.MsgTx:
			p.OnTx(msg)
		//case *payload.MsgBinary:
		//	p.OnBinary(msg)
		//case *payload.MsgCandidate:
		//	p.OnCandidate(msg)
		//case *payload.MsgCertificate:
		//	p.OnCertificate(msg)
		//case *payload.MsgCertificateReq:
		//	p.OnCertificateReq(msg)
		//case *payload.MsgMemPool:
		//	p.OnMemPool(msg)
		case *payload.MsgNotFound:
			p.OnNotFound(msg)
		//case *payload.MsgPing:
		//	p.OnPing(msg)
		//case *payload.MsgPong:
		//	p.OnPong(msg)
		//case *payload.MsgReduction:
		//	p.OnReduction(msg)
		case *payload.MsgReject:
			p.OnReject(msg)
		//case *payload.MsgScore:
		//	p.OnScore(msg)
		default:
			log.WithField("prefix", "peer").Warnf("Did not recognise message '%s'", msg.Command()) //Do not disconnect peer, just log message
		}
	}

	idleTimer.Stop()
	p.Disconnect()
}

// WriteLoop will queue all messages to be written to the peer
func (p *Peer) WriteLoop() {
	for atomic.LoadInt32(&p.disconnected) == 0 {
		select {
		case f := <-p.outch:
			f()
		case <-p.Detector.Quitch: // if the detector quits, disconnect peer
			p.Disconnect()
		}
	}
}

/**
 * Received data requests from other peers
 */

// OnGetData Listener. Is called after receiving a 'getdata' msg
func (p *Peer) OnGetData(msg *payload.MsgGetData) {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.GetData, p.addr)
	p.inch <- func() {
		if p.hndlr.OnGetData != nil {
			p.hndlr.OnGetData(p, msg)
		}
	}
}

// OnTx Listener. Is called after receiving a 'tx' msg
func (p *Peer) OnTx(msg *payload.MsgTx) {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.Tx, p.addr)
	p.inch <- func() {
		if p.hndlr.OnTx != nil {
			p.hndlr.OnTx(p, msg)
		}
	}
}

// OnInv Listener. Is called after receiving a 'inv' msg
// It could be received in reply to 'getblocks' or unsolicited.
// In the last case we have to check if we already have the tx/block or not.
// We need to send a 'getdata' msg to receive the actual tx/block(s).
func (p *Peer) OnInv(msg *payload.MsgInv) {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.Inv, p.addr)
	p.inch <- func() {
		// TODO: Check if we are interested or always get the data?
		getdata := payload.NewMsgGetData()
		for _, vector := range msg.Vectors {
			switch vector.Type {
			case payload.InvTx:
				tx := transactions.NewTX()
				tx.Hash = vector.Hash
				getdata.AddTx(tx)
			case payload.InvBlock:
				block := payload.NewBlock()
				block.Header.Hash = vector.Hash
				getdata.AddBlock(block)
			default:
				log.WithField("prefix", "peer").Errorf("Unknown InvType in '%s' msg from %s", commands.Inv, p.addr)
			}
		}
		p.Write(getdata)
	}
}

// OnGetHeaders Listener, outside of the anonymous func will be extra functionality like timing
func (p *Peer) OnGetHeaders(msg *payload.MsgGetHeaders) {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.GetHeaders, p.addr)
	p.inch <- func() {
		if p.hndlr.OnGetHeaders != nil {
			p.hndlr.OnGetHeaders(p, msg)
		}
	}
}

// OnAddr Listener. Is called after receiving a 'addr' msg
func (p *Peer) OnAddr(msg *payload.MsgAddr) {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.Addr, p.addr)
	p.inch <- func() {
		if p.hndlr.OnAddr != nil {
			p.hndlr.OnAddr(p, msg)
		}
	}
}

// OnGetAddr Listener. Is called after receiving a 'getaddr' msg
func (p *Peer) OnGetAddr(msg *payload.MsgGetAddr) {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.GetAddr, p.addr)
	p.inch <- func() {
		if p.hndlr.OnGetAddr != nil {
			p.hndlr.OnGetAddr(p, msg)
		}
	}
}

// OnGetBlocks Listener. Is called after receiving a 'getblocks' msg
func (p *Peer) OnGetBlocks(msg *payload.MsgGetBlocks) {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.GetBlocks, p.addr)
	p.inch <- func() {
		if p.hndlr.OnGetBlocks != nil {
			p.hndlr.OnGetBlocks(p, msg)
		}
	}
}

// OnBlock Listener. Is called after receiving a 'blocks' msg
func (p *Peer) OnBlock(msg *payload.MsgBlock) {
	log.WithField("prefix", "peer").Infof("Received a '%s' message (Height=%d) from %s", commands.Block, msg.Block.Header.Height, p.addr)
	p.inch <- func() {
		if p.hndlr.OnBlock != nil {
			p.hndlr.OnBlock(p, msg)
		}
	}
}

// OnVersion Listener will be called during the handshake, any error checking should be done here for 'version' msg.
// This should only ever be called during the handshake. Any other place and the peer will disconnect.
func (p *Peer) OnVersion(msg *payload.MsgVersion) error {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.Version, p.addr)
	if msg.Nonce == p.Nonce {
		log.WithField("prefix", "peer").Infof("Received '%s' message from yourself", commands.Version)
		p.conn.Close()
		return errors.New("Self connection, disconnecting Peer")
	}

	if protocol.ProtocolVersion != msg.Version {
		log.WithField("prefix", "peer").Infof("Received an invalid '%s' message from %s\n"+
			"Dusk backwards compatible Version Management not implemented yet", commands.Version, p.addr)
		rejectMsg := payload.NewMsgReject(string(commands.Version), payload.RejectInvalid, "invalid")
		return p.Write(rejectMsg)
	}

	p.versionKnown = true
	p.protoVer = msg.Version
	//p.services = msg.Services
	//p.userAgent = string(msg.UserAgent)
	//p.relay = msg.Relay
	p.createdAt = time.Now()
	return nil
}

// OnVerack Listener will be called during the handshake.
// This should only ever be called during the handshake. Any other place and the peer will disconnect.
func (p *Peer) OnVerack(msg *payload.MsgVerAck) error {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.VerAck, p.addr)
	return nil
}

// OnHeaders Listener. Is called after receiving a 'headers' msg
func (p *Peer) OnHeaders(msg *payload.MsgHeaders) {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.Headers, p.addr)
	p.inch <- func() {
		if p.hndlr.OnHeaders != nil {
			p.hndlr.OnHeaders(p, msg)
		}
	}
}

// OnBinary Listener. Is called after receiving a 'binary' msg
func (p *Peer) OnBinary(msg *payload.MsgBinary) {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.Binary, p.addr)
	p.inch <- func() {
		if p.hndlr.OnBinary != nil {
			p.hndlr.OnBinary(p, msg)
		}
	}
}

// OnCertificate Listener. Is called after receiving a 'certificate' msg
func (p *Peer) OnCertificate(msg *payload.MsgCertificate) {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.Certificate, p.addr)
	p.inch <- func() {
		if p.hndlr.OnCertificate != nil {
			p.hndlr.OnCertificate(p, msg)
		}
	}
}

// OnCertificateReq Listener. Is called after receiving a 'certificatereq' msg
func (p *Peer) OnCertificateReq(msg *payload.MsgCertificateReq) {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.CertificateReq, p.addr)
	p.inch <- func() {
		if p.hndlr.OnCertificateReq != nil {
			p.hndlr.OnCertificateReq(p, msg)
		}
	}
}

// OnMemPool Listener. Is called after receiving a 'mempool' msg
func (p *Peer) OnMemPool(msg *payload.MsgMemPool) {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.MemPool, p.addr)
	p.inch <- func() {
		if p.hndlr.OnMemPool != nil {
			p.hndlr.OnMemPool(p, msg)
		}
	}
}

// OnNotFound Listener. Is called after receiving a 'notfound' msg
func (p *Peer) OnNotFound(msg *payload.MsgNotFound) {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.NotFound, p.addr)

	// Remove the message we initially requested from the Detector
	//TODO: Make a separate function and check whether we need more payload.InvXxx types (e.g. InvHdr)
	for _, vector := range msg.Vectors {
		if vector.Type == payload.InvBlock {
			p.Detector.RemoveMessage(commands.GetHeaders)
		} else {
			p.Detector.RemoveMessage(commands.Tx)
		}
	}
	p.inch <- func() {
		if p.hndlr.OnNotFound != nil {
			p.hndlr.OnNotFound(p, msg)
		}
	}
}

// OnPing Listener. Is called after receiving a 'ping' msg
func (p *Peer) OnPing(msg *payload.MsgPing) {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.Ping, p.addr)
	p.inch <- func() {
		if p.hndlr.OnPing != nil {
			p.hndlr.OnPing(p, msg)
		}
	}
}

// OnPong Listener. Is called after receiving a 'pong' msg
func (p *Peer) OnPong(msg *payload.MsgPong) {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.Pong, p.addr)
	p.inch <- func() {
		if p.hndlr.OnPong != nil {
			p.hndlr.OnPong(p, msg)
		}
	}
}

// OnReduction Listener. Is called after receiving a 'reduction' msg
func (p *Peer) OnReduction(msg *payload.MsgReduction) {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.Reduction, p.addr)
	p.inch <- func() {
		if p.hndlr.OnReduction != nil {
			p.hndlr.OnReduction(p, msg)
		}
	}
}

// OnReject Listener. Is called after receiving a 'reject' msg
func (p *Peer) OnReject(msg *payload.MsgReject) {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.Reject, p.addr)
	p.inch <- func() {
		if p.hndlr.OnReject != nil {
			p.hndlr.OnReject(p, msg)
		}
	}
}

// OnScore Listener. Is called after receiving a 'score' msg
func (p *Peer) OnScore(msg *payload.MsgScore) {
	log.WithField("prefix", "peer").Infof(receivedMessageFromStr, commands.Score, p.addr)
	p.inch <- func() {
		if p.hndlr.OnScore != nil {
			p.hndlr.OnScore(p, msg)
		}
	}
}

/**
 * Requesting data from other peers
 */

// RequestHeaders will ask a peer for headers.
func (p *Peer) RequestHeaders(hash []byte) error {
	log.WithField("prefix", "peer").Infof("Sending '%s' msg requesting headers from %s", commands.GetHeaders, p.addr)
	c := make(chan error)
	p.outch <- func() {
		p.Detector.AddMessage(commands.GetHeaders)
		stop := make([]byte, 32)
		getHeaders := payload.NewMsgGetHeaders(hash, stop)
		err := p.Write(getHeaders)
		c <- err
	}

	return <-c
}

// RequestTx will ask a peer for a transaction.
// It will put a function on the outgoing peer queue to send a 'getdata' msg
// to an other peer. An error from this function will return this error from RequestTx.
func (p *Peer) RequestTx(tx transactions.Stealth) error {
	log.WithField("prefix", "peer").Infof("Sending '%s' msg, requesting transactions from %s", commands.GetData, p.addr)
	c := make(chan error)

	p.outch <- func() {
		p.Detector.AddMessage(commands.GetData)
		getdata := payload.NewMsgGetData()
		getdata.AddTx(&tx)
		err := p.Write(getdata)
		c <- err
	}

	return <-c
}

// RequestBlocks will ask a peer for blocks.
// It will put a function on the outgoing peer queue to send a 'getdata' msg to an other peer.
// The same possible function error will be returned from this method.
func (p *Peer) RequestBlocks(hashes [][]byte) error {
	log.WithField("prefix", "peer").Infof("Sending '%s' msg, requesting blocks from %s", commands.GetData, p.addr)
	c := make(chan error)

	blocks := make([]*payload.Block, 0, len(hashes))
	for _, hash := range hashes {
		// Create a block from requested hash
		block := payload.NewBlock()
		block.Header.Hash = hash
		blocks = append(blocks, block)
	}

	p.outch <- func() {
		p.Detector.AddMessage(commands.GetData)
		getdata := payload.NewMsgGetData()
		getdata.AddBlocks(blocks)
		err := p.Write(getdata)
		c <- err
	}

	return <-c
}

// RequestAddresses will ask a peer for addresses.
// It will put a function on the outgoing peer queue to send a 'getaddr' msg to an other peer.
// The same possible function error will be returned from this method.
func (p *Peer) RequestAddresses() error {
	log.WithField("prefix", "peer").Infof("Sending '%s' msg, requesting addresses from %s", commands.GetAddr, p.addr)
	c := make(chan error)

	p.outch <- func() {
		p.Detector.AddMessage(commands.GetAddr)
		getaddr := payload.NewMsgGetAddr()
		err := p.Write(getaddr)
		c <- err
	}

	return <-c
}
