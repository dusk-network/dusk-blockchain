// The sync manager will use a modified version of the initial block download in bitcoin
// Seen here: https://en.bitcoinwiki.org/wiki/Bitcoin_Core_0.11_(ch_5):_Initial_Block_Download
// MovingWindow is a desired featured from the original codebase

package syncmgr

import (
	"encoding/hex"
	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"golang.org/x/sync/syncmap"
	"net"
)

var (
	// This is the maximum amount of inflight objects that we would like to have
	// Number taken from original codebase
	maxBlockRequest = 1024

	// This is the maximum amount of blocks that we will ask for from a single peer
	// Number taken from original codebase
	maxBlockRequestPerPeer = 16
)

// Syncmgr holds pointers to peer- and address-manager and keeps the state of
// synchronisation of headers and blocks
type Syncmgr struct {
	config            Config
	pcfg              peermgr.ResponseHandler
	Mode              int // 1 = headersFirst, 2 = Blocks, 3 = Maintain
	headers           [][]byte
	inflightBlockReqs syncmap.Map // When we send a req for block, we will put hash in here, along with peer who we requested it from
}

// New returns a Synchronisation Manager
func New(cfg Config) (*Syncmgr, error) {
	var err error
	sm := &Syncmgr{
		config:  cfg,
		Mode:    1,
		headers: [][]byte{},
	}
	sm.pcfg = sm.setupPeerResponseHandler()

	if err != nil {
		return nil, err
	}
	return sm, nil
}

func (s *Syncmgr) setupPeerResponseHandler() peermgr.ResponseHandler {
	peerRspHndlr := peermgr.ResponseHandler{
		OnHeaders: s.OnHeaders,
		//OnNotFound:	      s.OnNotFound,
		OnGetData: s.OnGetData,
		//OnTx:	          s.OnTx,
		OnGetHeaders: s.OnGetHeaders,
		//OnAddr:           s.OnAddr,
		OnGetAddr:   s.OnGetAddr,
		OnGetBlocks: s.OnGetBlocks,
		OnBlock:     s.OnBlock,
		//OnBinary:         s.OnBinary,
		//OnCandidate:      s.OnCandidate,
		//OnCertificate:    s.OnCertificate,
		//OnCertificateReq: s.OnCertificateReq,
		OnMemPool: s.OnMemPool,
		//OnPing:           s.OnPing,
		//OnPong:           s.OnPong,
		//OnReduction:      s.OnReduction,
		OnReject: s.OnReject,
		//OnScore:          s.OnScore
		//OnInv:            s.OnInv
	}

	return peerRspHndlr
}

// CreatePeer is called after a connection to a peer was successful
func (s *Syncmgr) CreatePeer(con net.Conn, inbound bool) *peermgr.Peer {
	p := peermgr.NewPeer(con, inbound, s.pcfg)
	s.addPeer(p)

	// TODO: set the unchangeable states
	return p
}

// addPeer adds a peer for the peer manager to use
func (s *Syncmgr) addPeer(peer *peermgr.Peer) {
	s.config.AddPeer(peer)
}

// OnGetHeaders receives 'getheaders' msgs from a peer, reads them from the chain db
// and sends them to the requesting peer.
func (s *Syncmgr) OnGetHeaders(p *peermgr.Peer, msg *payload.MsgGetHeaders) {
	log.WithField("prefix", "syncmgr").Debug("Syncmgr OnGetHeaders called")
	// The caller peer wants some headers from our blockchain.
	msgHeaders, err := s.getHeaders(msg)
	if err == nil {
		p.Write(msgHeaders)
	} else {
		log.WithField("prefix", "syncmgr").Errorf("Failed to send '%s' to requesting peer %s: %s", topics.Headers, p.RemoteAddr().String(), err)
	}
}

// OnHeaders receives 'headers' msgs from an other peer and adds them to the chain.
func (s *Syncmgr) OnHeaders(p *peermgr.Peer, msg *payload.MsgHeaders) {

	log.WithField("prefix", "syncmgr").Debug("Sync manager OnHeaders called")

	// Any headers received?
	if len(msg.Headers) < 1 {
		log.WithField("prefix", "syncmgr").Infof("'%s' msg is empty", topics.Headers)
		return
	}

	// On receipt of Headers check what mode we are in.
	// HeadersMode, we check if there is 2k. If so call again. If not then change mode into BlocksOnly
	if s.Mode == 1 {
		err := s.HeadersFirstMode(p, msg)
		if err != nil {
			log.WithField("prefix", "syncmgr").Error("Failed to read block headers:", err)
			return // TODO:We should custom name error so, that we can do something on WrongHash Error, Peer disconnect error
		}
		return
	}
}

// HeadersFirstMode receives 'headers' msgs from an other peer and adds them to the chain.
func (s *Syncmgr) HeadersFirstMode(p *peermgr.Peer, msg *payload.MsgHeaders) error {
	log.WithField("prefix", "syncmgr").Debug("Headers first mode")

	// Validate Headers
	if err := s.config.ValidateHeaders(msg); err != nil {
		// Re-request headers from a different peer
		s.config.Disconnect(p)
		log.WithField("prefix", "syncmgr").Error("Failed to validate headers:", err)
		return err
	}

	// Persist Headers to chain
	if err := s.config.AddHeaders(msg); err != nil {
		log.WithField("prefix", "syncmgr").Error("Failed to add headers", err)
		return err
	}

	// Add header hashes into slice
	// Request first batch of blocks here
	hashes := make([][]byte, 0, len(msg.Headers))
	for _, header := range msg.Headers {
		hashes = append(hashes, header.Hash)
	}
	s.headers = append(s.headers, hashes...)

	if len(msg.Headers) <= 2*1e3 { // Should be less than 2000, leave it as this for tests
		log.WithField("prefix", "syncmgr").Debug("Switching to BlocksOnly Mode")
		s.Mode = 2 // Switch to BlocksOnly. XXX: because HeadersFirst is not in parallel, no race condition here.
		for len(s.headers) > 0 {
			if err := s.RequestMoreBlocks(); err != nil {
				log.WithField("prefix", "syncmgr").Error("Failed to request blocks from peer in 'BlocksOnly' mode")
				return err
			}
		}
	}
	latestHeader := msg.Headers[len(msg.Headers)-1]
	_, err := s.config.RequestHeaders(latestHeader.Hash)
	s.Mode = 1
	return err
}

// RequestMoreBlocks request blocks from an other peer and keeps an admin of the requested blocks and peers.
func (s *Syncmgr) RequestMoreBlocks() error {
	var blockReq [][]byte
	var reqAmount int

	if len(s.headers) >= maxBlockRequestPerPeer {
		reqAmount = maxBlockRequestPerPeer
	} else {
		reqAmount = len(s.headers)
	}
	blockReq = s.headers[:reqAmount]

	peer, err := s.config.RequestBlocks(blockReq)
	if err != nil { // This could happen if the peermanager has no valid peers to connect to. We should wait a bit and re-request
		return err // alternatively we could make RequestBlocks blocking, then make sure it is not triggered when a block is received
	}

	for _, hash := range s.headers {
		hashKey := hex.EncodeToString(hash)
		s.inflightBlockReqs.Store(hashKey, peer)
	}
	s.headers = s.headers[reqAmount:]
	// NONONO: Here we do not pass all of the hashes to peermanager because
	// it is not the peermanagers responsibility to manage inflight blocks
	return err
}

// RequestAddresses request addresses from an other peer
func (s *Syncmgr) RequestAddresses() error {
	return s.config.RequestAddresses()
}

// OnInv receives 'inv' msgs from a peer.
func (s *Syncmgr) OnInv(p *peermgr.Peer, msg *payload.MsgGetData) {
	//TODO
}

// OnGetData receives 'getdata' msgs from a peer.
// This could be a request for a specific Tx or Block and will be read from the chain db.
// and send to the requesting peer.
func (s *Syncmgr) OnGetData(p *peermgr.Peer, msg *payload.MsgGetData) {
	log.WithField("prefix", "syncmgr").Debug("Syncmgr OnGetData called")
	// The caller peer wants some txs and/or blocks from our blockchain.
	for _, vector := range msg.Vectors {
		switch vector.Type {
		case payload.InvTx:
			//tx := transactions.NewTX()
			//tx.Hash = vector.Hash
			//s.chain.GetTx()
		case payload.InvBlock:
			block, err := s.config.GetBlock(vector.Hash)
			if err == nil {
				blockMsg := payload.NewMsgBlock(block)
				p.Write(blockMsg)
			} else {
				log.WithField("prefix", "syncmgr").Fatalf("Failed to read Block by hash %x", vector.Hash)
			}
		default:
			log.WithField("prefix", "syncmgr").Errorf("Unknown InvType in '%s' msg from %s", topics.Inv, p.RemoteAddr().String())
		}
	}
}

// OnGetBlocks receives 'getblocks' msg from a peer, then passes it to the blockchain to process.
func (s *Syncmgr) OnGetBlocks(p *peermgr.Peer, msg *payload.MsgGetBlocks) {
	log.WithField("prefix", "syncmgr").Debugf("Received a '%s' msg from %s", topics.GetBlocks, p.RemoteAddr().String())
	//TODO
}

// OnBlock receives a 'block' msg from a peer, then passes it to the blockchain to process
func (s *Syncmgr) OnBlock(p *peermgr.Peer, msg *payload.MsgBlock) {
	err := s.config.AcceptBlock(msg.Block)
	if err != nil {
		// Put headers back in front of queue to fetch block for.
		log.WithField("prefix", "syncmgr").Error(err)
	}
}

// OnGetAddr receives a getaddr msg from a peer, then passes it to the Address Manager to process.
func (s *Syncmgr) OnGetAddr(p *peermgr.Peer, msg *payload.MsgGetAddr) {
	addresses := s.config.GetGoodAddresses()
	addrMsg := payload.NewMsgAddr()
	for _, addr := range addresses {
		addrMsg.AddAddr(&addr)
	}
	// Push most recent peers to peer
	p.Write(addrMsg)
}

// OnAddr receives a addr msg from a peer, then passes it to the Address Manager to process.
func (s *Syncmgr) OnAddr(p *peermgr.Peer, msg *payload.MsgAddr) {
	s.config.AddAddrs(msg.Addresses)
}

// OnMemPool (TODO)
func (s *Syncmgr) OnMemPool(p *peermgr.Peer, msg *payload.MsgMemPool) {
	//err := s.chain.AddMempool(msg)
	//if err != nil {
	//	// Put headers back in front of queue to fetch block for.
	//	fmt.Println("Block had an error", err)
	//}
}

// OnReject (TODO)
func (s *Syncmgr) OnReject(p *peermgr.Peer, msg *payload.MsgReject) {

}
