package noded

import (
	log "github.com/sirupsen/logrus"
	cnf "github.com/spf13/viper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/noded/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/connmgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/util"
)

// NodeServer acts as P2P server and sets up the blockchain.
// Next to that it initializes a set of managers:
// - Connection manager: manages peer connections
// - Synchronization manager: manages the synchronization of peers via messages
type NodeServer struct{}

func (s *NodeServer) setupChain() error {
	// Blockchain - Integrate
	_, err := core.GetBcInstance()
	return err
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

	if err := server.setupChain(); err != nil {
		log.WithField("prefix", "noded").Errorf("Failed to setup the Blockchain: %s", err)
		done <- true
	}

	server.Run()

	ip, _ := util.GetOutboundIP()

	nodeAddr := payload.NetAddress{ip, uint16(cnf.GetInt("net.peer.port"))}
	log.WithField("prefix", "noded").Infof("Node server is running on %s", nodeAddr.String())

	// Prevent the server from stopping.
	// Server can only be stopped by 'done <- true' or Ctrl-C.
	<-done
}

// Run starts the server
func (s *NodeServer) Run() {
	// Run the Connection Manager
	cm := connmgr.GetInstance()
	cm.Run()
	// This is the first and only statically created request for a peer (other requests will be event driven)
	cm.NewRequest()
}
