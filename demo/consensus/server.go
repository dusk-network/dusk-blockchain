package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"sort"
	"sync"

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
	nonce uint64
	peers []*peermgr.Peer
	tx    *transactions.Stealth
	cfg   *peermgr.Config
	ctx   *user.Context
	wg    sync.WaitGroup
}

// OnAccept is the function that runs when a node tries to connect with us
func (s *Server) OnAccept(conn net.Conn) {
	fmt.Printf("someone has tried to connect to us, with the address %s\n", conn.RemoteAddr().String())

	// Add peer to server's list of peers
	peer := peermgr.NewPeer(conn, true, s.cfg)
	peer.Run()
	s.peers = append(s.peers, peer)

	// Send them our stake
	if err := peer.Write(payload.NewMsgTx(s.tx)); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	s.wg.Done()
}

// OnConnection is the function that runs when we connect to another node
func (s *Server) OnConnection(conn net.Conn, addr string) {
	fmt.Printf("we have connected to the node with the address %s\n", conn.RemoteAddr().String())

	// Add peer to server's list of peers
	peer := peermgr.NewPeer(conn, false, s.cfg)
	peer.Run()
	s.peers = append(s.peers, peer)

	// Send them our stake
	if err := peer.Write(payload.NewMsgTx(s.tx)); err != nil {
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

	fmt.Printf("[TX] Received a tx from node with address %s\n", peer.RemoteAddr().String())

	// Put information into our context object
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

	if err != nil {
		fmt.Println(err)
	}

	return nil
}
