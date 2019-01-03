package noded

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	cnf "github.com/spf13/viper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/blockchain"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/noded/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/connmgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/syncmgr"
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
	sm         *syncmgr.Syncmanager
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
		Port:         cnf.GetString("net.peer.port"),
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
			return errors.New("Failed to get LatestHeader:" + err.Error())
		}
	} else { //TODO: Is this true?
		return errors.New("Failed to add genesis block")
	}

	return err
}

func (s *NodeServer) setupSyncManager() error {
	// Sync Manager - Integrate
	s.sm = syncmgr.New(syncmgr.Config{
		Chain:    s.chain,
		BestHash: s.latestHash,
	})
	return nil
}

func (s *NodeServer) setupPeerConfig() error {
	// Peer config struct - Integrate
	s.peercfg = peer.LocalConfig{
		Net:         protocol.MainNet,
		UserAgent:   protocol.UserAgent,
		Services:    protocol.NodePeerService,
		Nonce:       1200,
		ProtocolVer: protocol.ProtocolVersion,
		Relay:       false,
		Port:        uint16(cnf.GetInt("net.peer.port")),
		StartHeight: LocalHeight,
		// TODO: Discuss if ALL msgs should be handled via the syncmgr?
		OnGetHeaders: s.sm.OnGetHeaders,
		OnHeaders:    s.sm.OnHeaders,
		OnGetData:    s.sm.OnGetData,
		OnBlock:      s.sm.OnBlock,
		OnAddr:       s.sm.OnAddr,
		OnMemPool:    s.sm.OnMemPool,
	}
	return nil
}

// Run starts the server and adds the necessary configurations
func (s *NodeServer) Run() error {

	// Add all other run based methods for modules

	// Connmgr - Run
	s.cm.Run()

	//Iterate through seed list and connect to first available one
	// TODO: Find out if one peer sync is enough or try all seeds to become fully synced to tip of chain
	ip, _ := util.GetOutboundIP()
	toAddr := ip.String() + ":10111"
	err := s.cm.Connect(&connmgr.Request{
		Addr: toAddr,
	})
	if err != nil {
		return err
	}

	addr := payload.NetAddress{ip, s.peercfg.Port}
	log.WithField("prefix", "noded").Infof("Node server is running on %s", addr.String())

	return err
}

// Start loads the configuration, sets up a set of managers that together control the server
// and runs the daemon process.
func Start() {
	config.LoadConfig()
	setup()
}

func setup() {

	server := NodeServer{}
	err := server.setupConnMgr()
	if err != nil {
		log.WithField("prefix", "noded").Error("Failed to setup the Connection Manager:", err)
		os.Exit(0) //TODO: Exit gracefully (done channel ??)
	}

	err = server.setupChain()
	if err != nil {
		log.WithField("prefix", "noded").Error("Failed to setup the Blockchain:", err)
		os.Exit(0) //TODO: Exit gracefully (done channel ??)
	}

	err = server.setupSyncManager()
	if err != nil {
		log.WithField("prefix", "noded").Error("Failed to setup the Synchronisation Manager:", err)
		os.Exit(0) //TODO: Exit gracefully (done channel ??)
	}

	err = server.setupPeerConfig()
	if err != nil {
		log.WithField("prefix", "noded").Error("Failed to setup the Peer configuration:", err)
		os.Exit(0) //TODO: Exit gracefully (done channel ??)
	}

	err = server.Run()
	if err != nil {
		log.WithField("prefix", "noded").Error(err)
		os.Exit(0) //TODO: Exit gracefully (done channel ??)
	}

	// Block the channel to prevent the server from stopping.
	// It's anonymous so server can only be stopped by Ctrl-C.
	// TODO: Use a named channel ('done') to gracefully shut down when setup failed.
	<-make(chan struct{})
}

// LocalHeight returns the current height of the localNode
func LocalHeight() uint64 {
	return 10
}

// GetAddress is not implemented yet
func (s *NodeServer) GetAddress() (string, error) {
	return "", nil
}

// OnConn is called when a successful outbound connection has been made
func (s *NodeServer) OnConn(conn net.Conn, addr string) {
	log.WithField("prefix", "noded").Infof("A connection to node %s was created", addr)

	p := peer.NewPeer(conn, false, s.peercfg)
	err := p.Run()

	if err != nil {
		log.WithField("prefix", "noded").Error("Failed to run peer:" + err.Error())
	}

	if err == nil {
		s.sm.AddPeer(p)
	}

	// TODO: This is here just to quickly test the system
	err = p.RequestHeaders(s.latestHash)
	log.WithField("prefix", "noded").Info("For tests, we are only fetching first 2k batch")
	if err != nil {
		log.WithField("prefix", "noded").Error(err.Error())
	}
}

// OnAccept is called when a successful inbound connection has been made
func (s *NodeServer) OnAccept(conn net.Conn) {
	log.WithField("prefix", "noded").Infof("Peer %s wants to connect", conn.RemoteAddr().String())

	p := peer.NewPeer(conn, true, s.peercfg)

	err := p.Run()
	if err != nil {
		fmt.Println("Error running peer " + err.Error())
	}

	if err == nil {
		s.sm.AddPeer(p)
	}

	//s.cm.NewRequest()
	log.WithField("prefix", "noded").Infof("Start listening for requests from node address %s", conn.RemoteAddr().String())
}
