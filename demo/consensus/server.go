package main

import (
	"fmt"
	"math/rand"
	"net"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

type Server struct {
	peers []*peermgr.Peer
}

func (s *Server) OnAccept(conn net.Conn) {
	fmt.Printf("someone has tried to connect to us, with the address %s \n", conn.RemoteAddr().String())

	// Add peer to server's list of peers
	peer := peermgr.NewPeer(conn, true, s.setupPeerConfig())
	s.peers = append(s.peers, peer)
}

func (s *Server) OnConnection(conn net.Conn, addr string) {
	fmt.Printf("we have connected to the node with the address %s \n", conn.RemoteAddr().String())

	// Add peer to server's list of peers
	peer := peermgr.NewPeer(conn, false, s.setupPeerConfig())
	s.peers = append(s.peers, peer)
}

func (s *Server) OnConsensus(peer *peermgr.Peer, msg *payload.MsgConsensus) {
	fmt.Printf("we have received a consensus message from peer %s , message type %d", peer.RemoteAddr().String(), msg.Payload.Type())
}

func (s *Server) setupPeerConfig() *peermgr.Config {

	hndlr := peermgr.ResponseHandler{
		OnHeaders:        nil,
		OnNotFound:       nil,
		OnGetData:        nil,
		OnTx:             nil,
		OnGetHeaders:     nil,
		OnAddr:           nil,
		OnGetAddr:        nil,
		OnGetBlocks:      nil,
		OnBlock:          nil,
		OnConsensus:      s.OnConsensus,
		OnCertificate:    nil,
		OnCertificateReq: nil,
		OnMemPool:        nil,
		OnPing:           nil,
		OnPong:           nil,
		OnReject:         nil,
		OnInv:            nil,
	}

	return &peermgr.Config{
		Magic:   protocol.TestNet,
		Nonce:   rand.Uint64(),
		Handler: hndlr,
	}
}
