package main

import (
	"fmt"
	"net"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

type Server struct {
	nonce     uint64
	peers     []*peermgr.Peer
	cfg       *peermgr.Config
	consensus *consensus.Consensus
}

func (s *Server) OnAccept(conn net.Conn) {
	fmt.Printf("someone has tried to connect to us, with the address %s \n", conn.RemoteAddr().String())

	// Add peer to server's list of peers
	peer := peermgr.NewPeer(conn, true, s.cfg)
	peer.Run()
	s.peers = append(s.peers, peer)
}

func (s *Server) OnConnection(conn net.Conn, addr string) {
	fmt.Printf("we have connected to the node with the address %s \n", conn.RemoteAddr().String())

	// Add peer to server's list of peers
	peer := peermgr.NewPeer(conn, false, s.cfg)
	peer.Run()
	s.peers = append(s.peers, peer)

	// Send a mock message upon connection
	// msg := mockConsensusMsg()
	// err := peer.Write(msg)
	// fmt.Println(err)
}

func OnConsensus(peer *peermgr.Peer, msg *payload.MsgConsensus) {
	fmt.Printf("we have received a consensus message from peer %s , message type %d\n", peer.RemoteAddr().String(), msg.Payload.Type())
}

func setupPeerConfig(nonce uint64) *peermgr.Config {
	handler := peermgr.ResponseHandler{
		OnHeaders:        nil,
		OnNotFound:       nil,
		OnGetData:        nil,
		OnTx:             nil,
		OnGetHeaders:     nil,
		OnAddr:           nil,
		OnGetAddr:        nil,
		OnGetBlocks:      nil,
		OnBlock:          nil,
		OnConsensus:      OnConsensus,
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
		Nonce:   nonce,
		Handler: handler,
	}
}

func (s *Server) sendMessage(magic protocol.Magic, p wire.Payload) error {
	for _, peer := range s.peers {
		fmt.Printf("writing a %s message to peer %s\n", p.Command(), peer.RemoteAddr().String())
		if err := peer.Write(p); err != nil {
			return err
		}
	}

	return nil
}
