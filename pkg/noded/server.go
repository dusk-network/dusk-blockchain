package noded

import (
	"errors"
	log "github.com/sirupsen/logrus"
	cnf "github.com/spf13/viper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/blockchain"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/noded/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/addrmgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/connmgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/syncmgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/util"
	"net"
	"strconv"
	"strings"
)

// NodeServer acts as P2P server and sets up the blockchain.
// Next to that it initializes a set of managers:
// - Connection manager: manages peer connections
// - Synchronization manager: manages the synchronization of peers via messages
type NodeServer struct {
	chain      *blockchain.Chain
	cm         *connmgr.Connmgr
	sm         *syncmgr.Syncmgr
	am         *addrmgr.Addrmgr
	peercfg    peer.LocalConfig
	latestHash []byte
}

func (s *NodeServer) setupConnMgr() error {
	// Connection Manager - Integrate
	s.cm = connmgr.New(connmgr.Config{
		GetAddress:   s.GetAddress,
		OnConnection: s.OnConn,
		OnFail:       s.OnFail,
		OnAccept:     s.OnAccept,
		OnMinGetAddr: s.OnMinGetAddr,
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

func (s *NodeServer) setupSyncMgr() error {
	// Sync Manager - Integrate
	s.sm = syncmgr.New(syncmgr.Config{
		Chain:    s.chain,
		BestHash: s.latestHash,
	})
	return nil
}

func (s *NodeServer) setupAddrMgr() {
	s.am = addrmgr.New()
	// Add the (permanent) seed addresses to the Address Manager
	var netAddrs []*payload.NetAddress
	addrs := cnf.GetStringSlice("net.peer.seeds")
	for _, addr := range addrs {
		s := strings.Split(addr, ":")
		port, _ := strconv.ParseUint(s[1], 10, 16)
		netAddrs = append(netAddrs, payload.NewNetAddress(s[0], uint16(port)))
	}
	s.am.AddAddrs(netAddrs)
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
		Port:         uint16(cnf.GetInt("net.peer.port")),
		StartHeight:  LocalHeight,
		OnGetHeaders: s.sm.OnGetHeaders,
		OnHeaders:    s.sm.OnHeaders,
		OnGetData:    s.sm.OnGetData,
		OnBlock:      s.sm.OnBlock,
		OnGetAddr:    s.am.OnGetAddr,
		OnAddr:       s.am.OnAddr,
		OnMemPool:    s.sm.OnMemPool,
	}
	return nil
}

// Start loads the configuration, sets up a set of managers that together control the server
// and runs the daemon process.
func Start() {
	config.LoadConfig()
	setup()
}

func setup() {
	done := make(chan bool, 1)

	server := NodeServer{}

	if err := server.setupConnMgr(); err != nil {
		log.WithField("prefix", "noded").Errorf("Failed to setup the Connection Manager: %s", err)
		done <- true
	}

	if err := server.setupChain(); err != nil {
		log.WithField("prefix", "noded").Errorf("Failed to setup the Blockchain: %s", err)
		done <- true
	}

	if err := server.setupSyncMgr(); err != nil {
		log.WithField("prefix", "noded").Errorf("Failed to setup the Synchronisation Manager: %s", err)
		done <- true
	}

	server.setupAddrMgr()

	if err := server.setupPeerConfig(); err != nil {
		log.WithField("prefix", "noded").Errorf("Failed to setup the Peer configuration: %s", err)
		done <- true
	}

	if err := server.Run(); err != nil {
		log.WithField("prefix", "noded").Error(err)
		done <- true
	} else {
		ip, _ := util.GetOutboundIP()
		nodeAddr := payload.NetAddress{ip, server.peercfg.Port}
		log.WithField("prefix", "noded").Infof("Node server is running on %s", nodeAddr.String())

		// TODO: This is here just to quickly test the system
		//	err = p.RequestHeaders(s.latestHash)
		//	log.WithField("prefix", "noded").Info("For tests, we are only fetching first 2k batch")
		//	if err != nil {
		//		log.WithField("prefix", "noded").Error(err.Error())
		//	}
	}

	// Prevent the server from stopping.
	// Server can only be stopped by 'done <- true' or Ctrl-C.
	<-done
}

// Run starts the server
// TODO: error?
func (s *NodeServer) Run() error {
	var err error

	// Run the Connection Manager
	s.cm.Run()
	// This is the first and only statically created request for a peer (other requests will be event driven)
	s.cm.NewRequest()

	return err
}

// LocalHeight returns the current height of the localNode
func LocalHeight() uint64 {
	return 10
}

// GetAddress gets a new address from Address Manager
func (s *NodeServer) GetAddress() (*payload.NetAddress, error) {
	return s.am.NewAddr()
}

// OnConn is called when a successful outbound connection has been made
func (s *NodeServer) OnConn(conn net.Conn, addr string) {
	log.WithField("prefix", "noded").Infof("A connection to node %s was created", addr)

	p := peer.NewPeer(conn, false, s.peercfg)
	err := p.Run()

	if err != nil {
		log.WithField("prefix", "noded").Errorf("Failed to run peer: %s", err.Error())
	}

	if err == nil {
		s.sm.AddPeer(p)
		s.am.ConnectionComplete(conn.RemoteAddr().String(), false)
	}

	// TODO: This is here just to quickly test the system
	err = p.RequestHeaders(s.latestHash)
	log.WithField("prefix", "noded").Info("For tests, we are only fetching first 2k batch")
	if err != nil {
		log.WithField("prefix", "noded").Error(err.Error())
	}
}

// OnFail is called when outbound connection failed
func (s *NodeServer) OnFail(addr string) {
	s.am.Failed(addr)
}

// OnMinGetAddr is called when the minimum of new addresses has exceeded
func (s *NodeServer) OnMinGetAddr() {
	if err := s.sm.RequestAddresses(); err != nil {
		log.WithField("prefix", "noded").Error("Failed to get addresses from peer after exceeding limit")
	}
}

// OnAccept is called when a successful inbound connection has been made
func (s *NodeServer) OnAccept(conn net.Conn) {
	log.WithField("prefix", "noded").Infof("Peer %s wants to connect", conn.RemoteAddr().String())

	p := peer.NewPeer(conn, true, s.peercfg)

	err := p.Run()
	if err != nil {
		log.WithField("prefix", "noded").Errorf("Failed to run peer: %s", err.Error())
	}

	if err == nil {
		s.sm.AddPeer(p)
		s.am.ConnectionComplete(conn.RemoteAddr().String(), true)
	}

	log.WithField("prefix", "noded").Infof("Start listening for requests from node address %s", conn.RemoteAddr().String())
}
