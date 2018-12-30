// This impl uses channels to simulate the queue handler with the actor model.
// A suitable number k ,should be set for channel size, because if #numOfMsg > k,
// we lose determinism. k chosen should be large enough that when filled, it shall indicate that
// the peer has stopped responding, since we do not have a pingMSG, we will need another way to shut down
// peers
package peer

import (
	"errors"
	"fmt"
	"net"
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

	// pingInterval = 20 * time.Second //Not implemented in neo clients
)

var (
	errHandShakeTimeout = errors.New("Handshake timed out, peers have " + string(handshakeTimeout) + " Seconds to Complete the handshake")
)

type Peer struct {
	config LocalConfig
	conn   net.Conn

	// Atomic vals
	disconnected int32

	// Unchangeable state: concurrent safe
	addr      string
	protoVer  uint32
	port      uint16
	inbound   bool
	userAgent string
	services  protocol.ServiceFlag
	createdAt time.Time
	relay     bool

	statemutex     sync.Mutex
	verackReceived bool
	versionKnown   bool

	*stall.Detector

	inch   chan func() // Will handle all incoming connections from peer
	outch  chan func() // Will handle all outcoming connections from peer
	quitch chan struct{}
}

func NewPeer(con net.Conn, inbound bool, cfg LocalConfig) *Peer {
	p := Peer{}
	p.inch = make(chan func(), inputBufferSize)
	p.outch = make(chan func(), outputBufferSize)
	p.quitch = make(chan struct{}, 1)
	p.inbound = inbound
	p.config = cfg
	p.conn = con
	p.createdAt = time.Now()
	p.addr = p.conn.RemoteAddr().String()

	p.Detector = stall.NewDetector(responseTime, tickerInterval)

	// TODO: set the unchangeable states
	return &p
}

// Write to a peer
func (p *Peer) Write(msg wire.Payload) error {
	return wire.WriteMessage(p.conn, p.config.Net, msg)
}

// Read from a peer
func (p *Peer) Read() (wire.Payload, error) {
	return wire.ReadMessage(p.conn, p.config.Net)
}

// Disconnects from a peer
func (p *Peer) Disconnect() {

	// return if already disconnected
	if atomic.LoadInt32(&p.disconnected) != 0 {
		return
	}

	atomic.AddInt32(&p.disconnected, 1)

	p.Detector.Quit()
	close(p.quitch)
	p.conn.Close()

	fmt.Println("Disconnected peer with address", p.RemoteAddr().String())
}

// Exposed API functions below
func (p *Peer) Port() uint16 {
	return p.port
}
func (p *Peer) CreatedAt() time.Time {
	return p.createdAt
}
func (p *Peer) CanRelay() bool {
	return p.relay
}
func (p *Peer) LocalAddr() net.Addr {
	return p.conn.LocalAddr()
}
func (p *Peer) RemoteAddr() net.Addr {
	return p.conn.RemoteAddr()
}
func (p *Peer) Services() protocol.ServiceFlag {
	return p.config.Services
}
func (p *Peer) Inbound() bool {
	return p.inbound
}
func (p *Peer) UserAgent() string {
	return p.config.UserAgent
}
func (p *Peer) IsVerackReceived() bool {
	return p.verackReceived
}
func (p *Peer) NotifyDisconnect() bool {
	fmt.Println("Peer has not disconnected yet")
	<-p.quitch
	fmt.Println("Peer has just disconnected")
	return true
}

//End of Exposed API functions//

// Ping not impl. in neo yet, adding it now
// will cause this client to disconnect from all other implementations
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

// run as a go-routine, will act as our queue for messages
// should be ran after handshake
func (p *Peer) StartProtocol() {
loop:
	for atomic.LoadInt32(&p.disconnected) == 0 {
		select {
		case f := <-p.inch:
			f()
		case <-p.quitch:
			break loop
		case <-p.Detector.Quitch:
			fmt.Printf("Peer %s stalled, disconnecting.\n", p.addr)
			break loop
		}
	}
	p.Disconnect()
}

// Should only be called after handshake is complete
// on a seperate go-routine.
// ReadLoop Will block on the read until a message is
// read

func (p *Peer) ReadLoop() {

	idleTimer := time.AfterFunc(idleTimeout, func() {
		fmt.Println("Timing out peer")
		p.Disconnect()
	})

loop:
	for atomic.LoadInt32(&p.disconnected) == 0 {

		idleTimer.Reset(idleTimeout) // reset timer on each loop

		readmsg, err := p.Read()

		// Message read; stop Timer
		idleTimer.Stop()

		if err != nil {
			fmt.Println("Err on read:", err) // This will also happen if Peer is disconnected
			break loop
		}

		// Remove message as pending from the stall detector
		p.Detector.RemoveMessage(readmsg.Command())

		switch msg := readmsg.(type) {

		case *payload.MsgVersion:
			fmt.Println("Already received a Version, disconnecting. " + p.RemoteAddr().String())
			break loop // We have already done the handshake, break loop and disconnect
		case *payload.MsgVerAck:
			if p.verackReceived {
				fmt.Println("Already received a Verack, disconnecting. " + p.RemoteAddr().String())
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
			p.OnBlocks(msg)
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
		case *payload.MsgBinary:
			p.OnBinary(msg)
		case *payload.MsgCandidate:
			p.OnCandidate(msg)
		case *payload.MsgCertificate:
			p.OnCertificate(msg)
		case *payload.MsgCertificateReq:
			p.OnCertificateReq(msg)
		case *payload.MsgMemPool:
			p.OnMemPool(msg)
		case *payload.MsgNotFound:
			p.OnNotFound(msg)
		case *payload.MsgPing:
			p.OnPing(msg)
		case *payload.MsgPong:
			p.OnPong(msg)
		case *payload.MsgReduction:
			p.OnReduction(msg)
		case *payload.MsgReject:
			p.OnReject(msg)
		case *payload.MsgScore:
			p.OnScore(msg)
		default:
			fmt.Println("Cannot recognise message", msg.Command()) //Do not disconnect peer, just Log Message
		}
	}

	idleTimer.Stop()
	p.Disconnect()
}

// WriteLoop will Queue all messages to be written to
// the peer.
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

func (p *Peer) OnGetData(msg *payload.MsgGetData) {

	p.inch <- func() {
		// fmt.Println(msg.Hashes)
		fmt.Printf("Received a 'getdata' message from %s.\n", p.addr)
	}
}

func (p *Peer) OnTx(msg *payload.MsgTx) {

	p.inch <- func() {
		// fmt.Println(msg.Hashes)
		getdata := payload.NewMsgGetData()
		id := msg.Tx
		getdata.AddTx(id)
		p.Write(getdata)
		fmt.Printf("Received a 'tx' message from %s.\n", p.addr)
	}
}

func (p *Peer) OnInv(msg *payload.MsgInv) {

	p.inch <- func() {
		if p.config.OnInv != nil {
			p.config.OnInv(p, msg)
		}
		fmt.Printf("Received an 'inv' message from %s.\n", p.addr)
	}
}

// OnGetHeaders Listener, outside of the anonymous func will be extra functionality
// like timing
func (p *Peer) OnGetHeaders(msg *payload.MsgGetHeaders) {
	p.inch <- func() {
		if p.config.OnGetHeaders != nil {
			p.config.OnGetHeaders(p, msg)
		}
		fmt.Printf("Received a 'getheaders' message from %s.\n", p.addr)
	}
}

// OnAddr Listener
func (p *Peer) OnAddr(msg *payload.MsgAddr) {
	p.inch <- func() {
		if p.config.OnAddr != nil {
			p.config.OnAddr(p, msg)
		}
		fmt.Printf("Received an 'addr' message from %s.\n", p.addr)
	}
}

// OnGetAddr Listener
func (p *Peer) OnGetAddr(msg *payload.MsgGetAddr) {
	p.inch <- func() {
		if p.config.OnGetAddr != nil {
			p.config.OnGetAddr(p, msg)
		}
		fmt.Printf("Received a 'getaddr' message from %s.\n", p.addr)
	}
}

// OnGetBlocks Listener
func (p *Peer) OnGetBlocks(msg *payload.MsgGetBlocks) {
	p.inch <- func() {
		if p.config.OnGetBlocks != nil {
			p.config.OnGetBlocks(msg)
		}
		fmt.Printf("Received a 'getblocks' message from %s.\n", p.addr)
	}
}

// OnBlocks Listener
func (p *Peer) OnBlocks(msg *payload.MsgBlock) {
	p.inch <- func() {
		if p.config.OnBlock != nil {
			p.config.OnBlock(p, msg)
		}
		fmt.Printf("Received a 'blocks' message from %s.\n", p.addr)
	}
}

// OnVersion Listener will be called during the handshake, any error checking should be done here for MsgVersion.
// This should only ever be called during the handshake. Any other place and the peer will disconnect.
func (p *Peer) OnVersion(msg *payload.MsgVersion) error {
	//if msg.Nonce == p.config.Nonce {
	//	p.conn.Close()
	//	return errors.New("Self connection, disconnecting Peer")
	//}
	//p.versionKnown = true
	//p.port = msg.Port
	//p.services = msg.Services
	//p.userAgent = string(msg.UserAgent)
	//p.createdAt = time.Now()
	//p.relay = msg.Relay
	return nil
}

// OnHeaders Listener
func (p *Peer) OnHeaders(msg *payload.MsgHeaders) {
	p.inch <- func() {
		if p.config.OnHeaders != nil {
			p.config.OnHeaders(p, msg)
		}
		fmt.Printf("Received a 'headers' message from %s.\n", p.addr)
	}
}

// OnBinary Listener
func (p *Peer) OnBinary(msg *payload.MsgBinary) {
	p.inch <- func() {
		if p.config.OnBinary != nil {
			p.config.OnBinary(p, msg)
		}
		fmt.Printf("Received a 'binary' message from %s.\n", p.addr)
	}
}

// OnCandidate Listener
func (p *Peer) OnCandidate(msg *payload.MsgCandidate) {
	p.inch <- func() {
		if p.config.OnCandidate != nil {
			p.config.OnCandidate(p, msg)
		}
		fmt.Printf("Received a 'candidate' message from %s.\n", p.addr)
	}
}

// OnCertificate Listener
func (p *Peer) OnCertificate(msg *payload.MsgCertificate) {
	p.inch <- func() {
		if p.config.OnCertificate != nil {
			p.config.OnCertificate(p, msg)
		}
		fmt.Printf("Received a 'certificate' message from %s.\n", p.addr)
	}
}

// OnCertificateReq Listener
func (p *Peer) OnCertificateReq(msg *payload.MsgCertificateReq) {
	p.inch <- func() {
		if p.config.OnCertificateReq != nil {
			p.config.OnCertificateReq(p, msg)
		}
		fmt.Printf("Received a 'certificatereq' message from %s.\n", p.addr)
	}
}

// OnMemPool Listener
func (p *Peer) OnMemPool(msg *payload.MsgMemPool) {
	p.inch <- func() {
		if p.config.OnMemPool != nil {
			p.config.OnMemPool(p, msg)
		}
		fmt.Printf("Received a 'mempool' message from %s.\n", p.addr)
	}
}

// OnNotFound Listener
func (p *Peer) OnNotFound(msg *payload.MsgNotFound) {
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
		if p.config.OnNotFound != nil {
			p.config.OnNotFound(p, msg)
		}
		fmt.Printf("Received a 'notfound' message from %s.\n", p.addr)
	}
}

// OnPing Listener
func (p *Peer) OnPing(msg *payload.MsgPing) {
	p.inch <- func() {
		if p.config.OnPing != nil {
			p.config.OnPing(p, msg)
		}
		fmt.Printf("Received a 'ping' message from %s.\n", p.addr)
	}
}

// OnPong Listener
func (p *Peer) OnPong(msg *payload.MsgPong) {
	p.inch <- func() {
		if p.config.OnPong != nil {
			p.config.OnPong(msg)
		}
		fmt.Printf("Received a 'pong' message from %s.\n", p.addr)
	}
}

// OnReduction Listener
func (p *Peer) OnReduction(msg *payload.MsgReduction) {
	p.inch <- func() {
		if p.config.OnReduction != nil {
			p.config.OnReduction(p, msg)
		}
		fmt.Printf("Received a 'reduction' message from %s.\n", p.addr)
	}
}

// OnReject Listener
func (p *Peer) OnReject(msg *payload.MsgReject) {
	p.inch <- func() {
		if p.config.OnReject != nil {
			p.config.OnReject(p, msg)
		}
		fmt.Printf("Received a 'reject' message from %s.\n", p.addr)
	}
}

// OnScore Listener
func (p *Peer) OnScore(msg *payload.MsgScore) {
	p.inch <- func() {
		if p.config.OnScore != nil {
			p.config.OnScore(p, msg)
		}
		fmt.Printf("Received a 'score' message from %s.\n", p.addr)
	}
}

// RequestHeaders will write a MsgGetHeaders to peer
func (p *Peer) RequestHeaders(hash []byte) error {
	fmt.Println("Sending 'getheaders' msg")
	c := make(chan error, 0)
	p.outch <- func() {
		p.Detector.AddMessage(commands.GetHeaders)
		stop := make([]byte, 32)
		getHeaders := payload.NewMsgGetHeaders(hash, stop)
		p.Write(getHeaders)
	}

	return <-c
}

// RequestTx will ask a peer for a transaction
func (p *Peer) RequestTx(tx transactions.Stealth) error {
	fmt.Println("Requesting tx from peer")
	c := make(chan error, 0)

	p.outch <- func() {
		p.Detector.AddMessage(commands.GetData)
		getdata := payload.NewMsgGetData()
		getdata.AddTx(&tx)
		err := p.Write(getdata)
		c <- err
	}

	return <-c
}

// RequestBlocks will ask a peer for blocks
func (p *Peer) RequestBlocks(hashes [][]byte) error {
	fmt.Println("Requesting block from peer")
	c := make(chan error, 0)

	blocks := make([]*payload.Block, len(hashes))
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
