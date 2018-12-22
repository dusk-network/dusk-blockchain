package main

import (
	"fmt"
	"net"

	//"gitlab.dusk.network/dusk-core/dusk-go/connmgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	//"gitlab.dusk.network/dusk-core/dusk-go/syncmanager"
)

// NodeServer acts as P2P server and sets up the blockchain and the database.
// Next to that it initializes a set of managers:
// - Connection manager: manages peer connections
// - Synchronization manager: manages the synchronization of peers via messages
// - Peer manager: acts as a middle manager and load balancer to be notified of added peers, data received etc.
type NodeServer struct {
	//chain *blockchain.Chain
	//db    *database.LDB // TODO(Kev) change to database.Database
	//sm    *syncmanager.Syncmanager
	//cm    *connmgr.Connmgr

	peercfg peer.LocalConfig

	latestHash []byte
}

func (s *NodeServer) setupConnMgr() error {
	// Connection Manager - Integrate
	//s.cm = connmgr.New(connmgr.Config{
	//	GetAddress:   nil,
	//	OnConnection: s.OnConn,
	//	OnAccept:     nil,
	//	Port:         "10333",
	//})

	return nil
}

//func (s *NodeServer) setupDatabase() error {
//	// Database -- Integrate
//	s.db = database.New("test")
//	return nil
//}

//func (s *NodeServer) setupChain() error {
//	// Blockchain - Integrate
//	s.chain = blockchain.New(s.db, protocol.MainNet)
//
//	if s.chain != nil {
//		table := database.NewTable(s.db, database.HEADER)
//		resa, err := table.Get(database.LATESTHEADER)
//		s.latestHash, err = util.Uint256DecodeBytes(resa)
//		if err != nil {
//			return errors.New("Failed to get LastHeader " + err.Error())
//		}
//	} else {
//		return errors.New("Failed to add genesis block")
//	}
//	return nil
//}

func (s *NodeServer) setupSyncManager() error {
	// Sync Manager - Integrate
	//s.sm = syncmanager.New(syncmanager.Config{
	//	//Chain:    s.chain,
	//	BestHash: s.latestHash,
	//})
	return nil
}

func (s *NodeServer) setupPeerConfig() error {
	// Peer config struct - Integrate
	s.peercfg = peer.LocalConfig{
		Net:         protocol.DevNet,
		UserAgent:   "DIG",
		Services:    protocol.NodePeerService,
		Nonce:       1200,
		ProtocolVer: 0,
		Relay:       false,
		Port:        10332,
		StartHeight: LocalHeight,
		//OnHeader:    s.sm.OnHeaders,
		//OnBlock:     s.sm.OnBlock,
		//OnMemPool:   s.sm.OnMemPool,
	}
	return nil
}

func (s *NodeServer) Run() error {

	// Add all other run based methods for modules

	// Connmgr - Run
	//s.cm.Run()
	//// Initial hardcoded nodes to connect to
	//err := s.cm.Connect(&connmgr.Request{
	//	Addr: "seed1.ngd.network:10333",
	//})
	return nil //err
}

func main() {
	setup()
}

func setup() {

	server := NodeServer{}
	//fmt.Println(server.sm)

	err := server.setupConnMgr()
	//err = server.setupDatabase()
	//err = server.setupChain()
	err = server.setupSyncManager()
	err = server.setupPeerConfig()

	//fmt.Println(server.sm)

	err = server.Run()
	if err != nil {
		fmt.Println(err)
	}

	<-make(chan struct{})

}

func LocalHeight() uint32 { // TODO: height could be endless. Use big.Int ??
	return 10
}

// OnConn is called when a successful connection has been made
func (s *NodeServer) OnConn(conn net.Conn, addr string) {
	fmt.Println(conn.RemoteAddr().String())
	fmt.Println(addr)

	p := peer.NewPeer(conn, false, s.peercfg)
	err := p.Run()

	if err != nil {
		fmt.Println("Error running peer" + err.Error())
	}

	if err == nil {
		//s.sm.AddPeer(p)
	}

	// This is here just to quickly test the system
	//err = p.RequestHeaders(s.latestHash)
	fmt.Println("For tests, we are only fetching first 2k batch")
	if err != nil {
		fmt.Println(err.Error())
	}
}
