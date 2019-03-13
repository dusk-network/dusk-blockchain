package noded

import (
	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/noded/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/noded/logging"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/addrmgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/connmgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/connmgr/connlstnr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/syncmgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/util"
)

const (
	configFile string = "cmd/noded/config/config.json"
)

// NodeServer acts as P2P server and sets up the blockchain and a set of managers.
// The managers form a set of controllers that each have responsibilities to keep
// the blockchain up to date.
type NodeServer struct {
	cm *connmgr.Connmgr
}

// Start loads the configuration, sets up a set of managers that together control
// the server and runs the daemon process.
func Start() {
	config.LoadConfig(configFile)
	setup()
}

func setup() {
	done := make(chan bool, 1)

	logging.ConfigureLogging()
	server := NodeServer{}

	am := server.setupAddrMgr()
	db, err := server.setupDatabase()
	bc, err := server.setupBlockchain(db)
	pm := server.setupPeerMgr()
	sm, err := server.setupSyncMgr(pm, am, bc)
	server.cm = server.setupConnMgr(sm, am, bc)

	if err != nil {
		log.WithField("prefix", "noded").Fatalf("Configuration of the node server failed:  %s", err.Error())
	}

	server.Run()

	ip, _ := util.GetOutboundIP()

	nodeAddr := payload.NetAddress{ip, config.EnvNetCfg.Peer.Port}
	log.WithField("prefix", "noded").Infof("Node server is running on %s", nodeAddr.String())

	// Prevent the server from stopping.
	// Server can only be stopped by 'done <- true' or Ctrl-C.
	<-done
}

func (s *NodeServer) setupAddrMgr() *addrmgr.Addrmgr {
	return addrmgr.New()
}

func (s *NodeServer) setupDatabase() (*database.BlockchainDB, error) {
	path := config.EnvNetCfg.Database.DirPath
	db, err := database.NewBlockchainDB(path)
	log.WithField("prefix", "database").Debugf("Path to database: %s", path)
	if err != nil {
		log.WithField("prefix", "database").Fatalf("Failed to find db path: %s", path)
	}
	return db, err
}

func (s *NodeServer) setupBlockchain(db *database.BlockchainDB) (*core.Blockchain, error) {
	bc, err := core.NewBlockchain(db, protocol.Magic(config.EnvNetCfg.Magic))
	return bc, err
}

func (s *NodeServer) setupPeerMgr() *peermgr.PeerMgr {
	return peermgr.New()
}

func (s *NodeServer) setupSyncMgr(pm *peermgr.PeerMgr, am *addrmgr.Addrmgr, bc *core.Blockchain) (*syncmgr.Syncmgr, error) {
	sm, err := syncmgr.New(syncmgr.Config{
		pm.AddPeer,
		pm.Disconnect,
		pm.RequestHeaders,
		pm.RequestBlocks,
		pm.RequestAddresses,
		am.GetGoodAddresses,
		am.AddAddrs,
		bc.ValidateHeaders,
		bc.AddHeaders,
		bc.GetBlock,
		bc.AcceptBlock,
		bc.GetHeaders,
	})
	return sm, err
}

func (s *NodeServer) setupConnMgr(sm *syncmgr.Syncmgr, am *addrmgr.Addrmgr, bc *core.Blockchain) *connmgr.Connmgr {
	cl := connlstnr.New(connlstnr.Config{
		sm.CreatePeer,
		sm.RequestAddresses,
		am.ConnectionComplete,
		am.NewAddres,
		am.Failed,
		bc.GetLatestHeader,
	})

	return connmgr.New(connmgr.Config{
		cl.GetAddress,
		cl.OnConnection,
		cl.OnAccept,
		cl.OnFail,
		cl.OnMinGetAddr,
	})
}

// Run starts the server
func (s *NodeServer) Run() {
	// Run the Connection Manager
	s.cm.Run()

	// This is the first and only statically created request for a peer (other requests will be event driven)
	s.cm.NewRequest()
}
