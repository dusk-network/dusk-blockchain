package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"os"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
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
	txs         []*transactions.Stealth
	cfg         *peermgr.Config
	ctx         *user.Context
	connectChan chan bool
}

// OnAccept is the function that runs when a node tries to connect with us
func (s *Server) OnAccept(conn net.Conn) {
	fmt.Printf("someone has tried to connect to us, with the address %s\n", conn.RemoteAddr().String())

	// Add peer to server's list of peers
	peer := peermgr.NewPeer(conn, true, s.cfg)
	peer.Run()
	s.peers = append(s.peers, peer)

	// Send them all our stakes
	for _, tx := range s.txs {
		if err := peer.Write(payload.NewMsgTx(tx)); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	s.connectChan <- true
}

// OnConnection is the function that runs when we connect to another node
func (s *Server) OnConnection(conn net.Conn, addr string) {
	fmt.Printf("we have connected to the node with the address %s\n", conn.RemoteAddr().String())

	// Add peer to server's list of peers
	peer := peermgr.NewPeer(conn, false, s.cfg)
	peer.Run()
	s.peers = append(s.peers, peer)

	// Send them our stake
	if err := peer.Write(payload.NewMsgTx(s.txs[0])); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	s.connectChan <- true
}

// OnConsensus is the function that runs when we receive a consensus message from another node
func (s *Server) OnConsensus(peer *peermgr.Peer, msg *payload.MsgConsensus) {
	switch msg.Payload.Type() {
	case consensusmsg.CandidateID:
		candMsg := msg.Payload.(*consensusmsg.Candidate)
		fmt.Printf("[CONSENSUS] Candidate block message\n\tBlock hash: %s\n", hex.EncodeToString(candMsg.Block.Header.Hash))
		s.ctx.CandidateChan <- msg
	case consensusmsg.CandidateScoreID:
		candScore := msg.Payload.(*consensusmsg.CandidateScore)
		fmt.Printf("[CONSENSUS] Candidate Score msg\n\tcandidate score: %d\n\tBlock hash :%s\n", candScore.Score, hex.EncodeToString(candScore.CandidateHash))
		s.ctx.CandidateScoreChan <- msg
	case consensusmsg.BlockReductionID:
		blockReduction := msg.Payload.(*consensusmsg.BlockReduction)
		fmt.Printf("[CONSENSUS] Block Reduction msg\n\tBlock hash :%s\n", hex.EncodeToString(blockReduction.BlockHash))
		if err := s.process(msg); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

// OnTx is the function that runs when we receive a transaction
// This is currently only used for stake transactions, so we will always infer
// that a received MsgTx contains a Stake TypeInfo.
func (s *Server) OnTx(peer *peermgr.Peer, msg *payload.MsgTx) {
	// Check if we have it already
	for _, tx := range s.txs {
		if tx == msg.Tx {
			return
		}
	}

	fmt.Printf("[TX] Received a tx from node with address %s\n", peer.RemoteAddr().String())
	s.txs = append(s.txs, msg.Tx)

	// Put information into our context object
	pl := msg.Tx.TypeInfo.(*transactions.Stake)
	s.ctx.Committee = append(s.ctx.Committee, pl.PubKeyEd)
	s.ctx.W += pl.Output.Amount
	pkEd := hex.EncodeToString(pl.PubKeyEd)
	s.ctx.NodeWeights[pkEd] = pl.Output.Amount
	pkBLS := hex.EncodeToString(pl.PubKeyBLS)
	s.ctx.NodeBLS[pkBLS] = pl.PubKeyEd
}

func setupPeerConfig(s *Server, nonce uint64) *peermgr.Config {
	handler := peermgr.ResponseHandler{
		OnHeaders:        nil,
		OnNotFound:       nil,
		OnGetData:        nil,
		OnTx:             s.OnTx,
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

func (s *Server) process(m *payload.MsgConsensus) error {
	err := msg.Process(s.ctx, m)
	if err != nil && err.Priority == prerror.High {
		return err.Err
	}

	return nil
}
