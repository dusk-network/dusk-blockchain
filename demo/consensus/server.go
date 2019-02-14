package main

import (
	"encoding/hex"
	"fmt"
	"net"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

// Server is the node server
type Server struct {
	nonce       uint64
	peers       []*peermgr.Peer
	cfg         *peermgr.Config
	ctx         *user.Context
	connectChan chan bool
}

// OnAccept is the function that runs when a node tries to connect with us
func (s *Server) OnAccept(conn net.Conn) {
	fmt.Printf("someone has tried to connect to us, with the address %s \n", conn.RemoteAddr().String())

	// Add peer to server's list of peers
	peer := peermgr.NewPeer(conn, true, s.cfg)
	peer.Run()
	s.peers = append(s.peers, peer)
	s.connectChan <- true
}

// OnConnection is the function that runs when we connect to another node
func (s *Server) OnConnection(conn net.Conn, addr string) {
	fmt.Printf("we have connected to the node with the address %s \n", conn.RemoteAddr().String())

	// Add peer to server's list of peers
	peer := peermgr.NewPeer(conn, false, s.cfg)
	peer.Run()
	s.peers = append(s.peers, peer)
	s.connectChan <- true
}

// OnConsensus is the function that runs when we receive a consensus message from another node
func (s *Server) OnConsensus(peer *peermgr.Peer, msg *payload.MsgConsensus) {
	switch msg.Payload.Type() {
	case consensusmsg.CandidateID:
		candMsg := msg.Payload.(*consensusmsg.Candidate)
		fmt.Printf("[CONSENSUS] Candidate block message\n Block hash: %s\n", hex.EncodeToString(candMsg.Block.Header.Hash))
		s.ctx.CandidateChan <- msg
	case consensusmsg.CandidateScoreID:
		candScore := msg.Payload.(*consensusmsg.CandidateScore)
		fmt.Printf("[CONSENSUS] Candidate Score msg\n candidate score: %d\n Block hash :%s\n", candScore.Score, hex.EncodeToString(candScore.CandidateHash))
		s.ctx.CandidateScoreChan <- msg
	}
}

func setupPeerConfig(s *Server, nonce uint64) *peermgr.Config {
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
		Nonce:   nonce,
		Handler: handler,
	}
}

func (s *Server) sendMessage(magic protocol.Magic, p wire.Payload) error {
	for _, peer := range s.peers {
		fmt.Printf("[CONSENSUS] writing message to peer %s\n", peer.RemoteAddr().String())
		if err := peer.Write(p); err != nil {
			return err
		}
	}

	return nil
}
