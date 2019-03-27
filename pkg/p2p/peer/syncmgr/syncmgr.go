// The sync manager will use a modified version of the initial block download in bitcoin
// Seen here: https://en.bitcoinwiki.org/wiki/Bitcoin_Core_0.11_(ch_5):_Initial_Block_Download
// MovingWindow is a desired featured from the original codebase

package syncmgr

import (
	"net"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermgr"
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
