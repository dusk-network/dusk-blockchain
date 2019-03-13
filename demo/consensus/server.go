package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"sort"
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"

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
	nonce   uint64
	port    string
	msgChan chan string
	peers   []*peermgr.Peer
	stake   *transactions.Stealth
	bid     *transactions.Stealth
	cfg     *peermgr.Config
	ctx     *user.Context
	wg      sync.WaitGroup
}

// OnAccept is the function that runs when a node tries to connect with us
func (s *Server) OnAccept(conn net.Conn) {
	// Add peer to server's list of peers
	peer := peermgr.NewPeer(conn, true, s.cfg)
	peer.Run()
	s.peers = append(s.peers, peer)

	// Send them our stake
	if err := peer.Write(payload.NewMsgTx(s.stake)); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Send them our bid
	if err := peer.Write(payload.NewMsgTx(s.bid)); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	s.wg.Done()
}

// OnConnection is the function that runs when we connect to another node
func (s *Server) OnConnection(conn net.Conn, addr string) {
	// Add peer to server's list of peers
	peer := peermgr.NewPeer(conn, false, s.cfg)
	peer.Run()
	s.peers = append(s.peers, peer)

	// Send them our stake
	if err := peer.Write(payload.NewMsgTx(s.stake)); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Send them our bid
	if err := peer.Write(payload.NewMsgTx(s.bid)); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	s.wg.Done()
}

// OnConsensus is the function that runs when we receive a consensus message from another node
func (s *Server) OnConsensus(peer *peermgr.Peer, msg *payload.MsgConsensus) {
	if err := s.process(msg); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// OnTx is the function that runs when we receive a transaction
// This is currently only used for stake transactions, so we will always infer
// that a received MsgTx contains a Stake TypeInfo.
func (s *Server) OnTx(peer *peermgr.Peer, msg *payload.MsgTx) {
	// Put information into our context object
	switch msg.Tx.Type {
	case transactions.StakeType:
		pl := msg.Tx.TypeInfo.(*transactions.Stake)
		s.ctx.Committee = append(s.ctx.Committee, pl.PubKeyEd)
		s.ctx.W += pl.Output.Amount
		pkEd := hex.EncodeToString(pl.PubKeyEd)
		s.ctx.NodeWeights[pkEd] = pl.Output.Amount
		pkBLS := hex.EncodeToString(pl.PubKeyBLS)
		s.ctx.NodeBLS[pkBLS] = pl.PubKeyEd

		// Sort committee
		sort.SliceStable(s.ctx.Committee, func(i int, j int) bool {
			return hex.EncodeToString(s.ctx.Committee[i]) < hex.EncodeToString(s.ctx.Committee[j])
		})
	case transactions.BidType:
		pl := msg.Tx.TypeInfo.(*transactions.Bid)
		s.ctx.SortedPubList = append(s.ctx.SortedPubList, pl.Secret)

		// Sort publist and put it into PubList
		sort.SliceStable(s.ctx.SortedPubList, func(i int, j int) bool {
			return hex.EncodeToString(s.ctx.SortedPubList[i]) < hex.EncodeToString(s.ctx.SortedPubList[j])
		})

		s.ctx.PubList = make([]byte, 0)
		for _, x := range s.ctx.SortedPubList {
			s.ctx.PubList = append(s.ctx.PubList, x...)
		}
	}
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
	var t string
	t = string(p.Command())

	if p.Command() == "consensus" {
		msg := p.(*payload.MsgConsensus)
		switch msg.ID {
		case consensusmsg.BlockAgreementID:
			t = "Block Agreement"
		case consensusmsg.BlockReductionID:
			t = "Block Reduction"
		case consensusmsg.CandidateID:
			t = "Block Candidate"
		case consensusmsg.CandidateScoreID:
			t = "Block Candidate Score"
		case consensusmsg.SigSetAgreementID:
			t = "Signature Set Agreement"
		case consensusmsg.SigSetCandidateID:
			t = "Signature Set Candidate"
		case consensusmsg.SigSetReductionID:
			t = "Signature Set Reduction"
		}
	}

	s.msgChan <- fmt.Sprintf("%s: sending %s message to committee\n", s.port, t)
	for _, peer := range s.peers {
		if err := peer.WriteConsensus(p); err != nil {
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
