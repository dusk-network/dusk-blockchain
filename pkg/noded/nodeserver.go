package main

import (
	"errors"
	"fmt"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/blockchain"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/connmgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/syncmanager"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/util"
	"net"
	"os"
)

// NodeServer acts as P2P server and sets up the blockchain.
// Next to that it initializes a set of managers:
// - Connection manager: manages peer connections
// - Synchronization manager: manages the synchronization of peers via messages
type NodeServer struct {
	chain      *blockchain.Chain
	sm         *syncmanager.Syncmanager
	cm         *connmgr.Connmgr
	peercfg    peer.LocalConfig
	latestHash []byte
}

func (s *NodeServer) setupConnMgr() error {
	// Connection Manager - Integrate
	s.cm = connmgr.New(connmgr.Config{
		GetAddress:   nil,
		OnConnection: s.OnConn,
		OnAccept:     s.OnAccept,
		Port:         "10333",
	})

	return nil
}

func (s *NodeServer) setupChain() error {
	// Blockchain - Integrate
	chain, err := blockchain.New(protocol.MainNet)
	s.chain = chain

	if s.chain != nil {
		s.latestHash, err = chain.GetLatestHeaderHash()
		if err != nil {
			return errors.New("Failed to get LatestHeader " + err.Error())
		}
	} else { //TODO: Is this true?
		return errors.New("Failed to add genesis block")
	}

	return err
}

func (s *NodeServer) setupSyncManager() error {
	// Sync Manager - Integrate
	s.sm = syncmanager.New(syncmanager.Config{
		Chain:    s.chain,
		BestHash: s.latestHash,
	})
	return nil
}

func (s *NodeServer) setupPeerConfig() error {
	// Peer config struct - Integrate
	s.peercfg = peer.LocalConfig{
		Net:          protocol.MainNet,
		UserAgent:    protocol.UserAgent,
		Services:     protocol.NodePeerService,
		Nonce:        1200,
		ProtocolVer:  protocol.ProtocolVersion,
		Relay:        false,
		Port:         10333,
		StartHeight:  LocalHeight,
		OnGetHeaders: s.sm.OnGetHeaders,
		OnHeaders:    s.sm.OnHeaders,
		OnBlock:      s.sm.OnBlock,
		OnMemPool:    s.sm.OnMemPool,
	}
	return nil
}

// Run starts the server and adds the necessary configurations
func (s *NodeServer) Run() error {

	// Add all other run based methods for modules

	// Connmgr - Run
	s.cm.Run()

	// Initial local hardcoded node to connect to
	ip, _ := util.GetOutboundIP()
	toAddr := ip.String() + ":10111"
	err := s.cm.Connect(&connmgr.Request{
		Addr: toAddr,
	})
	if err != nil {
		return err
	}

	addr := payload.NetAddress{ip, s.peercfg.Port}
	fmt.Printf("Node server is running on %s.\n", addr.String())

	return err
}

func main() {
	setup()
}

func setup() {

	server := NodeServer{}
	err := server.setupConnMgr()
	err = server.setupChain()
	if err != nil {
		fmt.Printf("Could not setup blockchain: %s\n", err)
		os.Exit(0) //TODO: Exit gracefully
	}

	err = server.setupSyncManager()
	err = server.setupPeerConfig()

	err = server.Run()
	if err != nil {
		fmt.Println(err)
	}

	<-make(chan struct{})
}

<<<<<<< HEAD
func LocalHeight() uint64 {
	//TODO: Read from database
=======
// LocalHeight returns the current height of the localNode
func LocalHeight() uint32 { // TODO: height could be endless. Use big.Int ??
>>>>>>> f69f9420bdac60c5179ec31d9a1cd0e7cf13d562
	return 10
}

func (s *NodeServer) GetAddress() (string, error) {
	return "", nil
}

// OnConn is called when a successful outbound connection has been made
func (s *NodeServer) OnConn(conn net.Conn, addr string) {
	fmt.Printf("A connection to node %s was created.\n", addr)

	p := peer.NewPeer(conn, false, s.peercfg)
	err := p.Run()

	if err != nil {
		fmt.Println("Error running peer " + err.Error())
	}

	if err == nil {
		s.sm.AddPeer(p)
	}

	// This is here just to quickly test the system
	err = p.RequestHeaders(s.latestHash)
	fmt.Println("For tests, we are only fetching first 2k batch")
	if err != nil {
		fmt.Println(err.Error())
	}
}

func (s *NodeServer) OnAccept(conn net.Conn) {
	fmt.Printf("Peer %s wants to connect.\n", conn.RemoteAddr().String())

	p := peer.NewPeer(conn, true, s.peercfg)

	err := p.Run()
	if err != nil {
		fmt.Println("Error running peer " + err.Error())
	}

	if err == nil {
		s.sm.AddPeer(p)
	}

	//s.cm.NewRequest()
	fmt.Printf("Start listening for requests from node address %s.\n", conn.RemoteAddr().String())
}

func IsTCPPortAvailable(port int) bool {
	ip, _ := util.GetOutboundIP()
	conn, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
