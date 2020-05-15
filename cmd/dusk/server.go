package main

import (
	"bytes"
	"github.com/dusk-network/dusk-blockchain/pkg/api"
	"net"

	"github.com/sirupsen/logrus"

	"github.com/dusk-network/dusk-blockchain/pkg/gql"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"

	"github.com/dusk-network/dusk-blockchain/pkg/core/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/chain"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/core/mempool"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactor"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
)

var logServer = logrus.WithField("process", "server")

// Server is the main process of the node
type Server struct {
	eventBus   *eventbus.EventBus
	rpcBus     *rpcbus.RPCBus
	loader     chain.Loader
	dupeMap    *dupemap.DupeMap
	counter    *chainsync.Counter
	gossip     *protocol.Gossip
	rpcWrapper *rpc.SrvWrapper
	// rpcClient     *rpc.Client
	cancelMonitor StopFunc
	kadPeer       *kadcast.Peer
}

// LaunchChain instantiates a chain.Loader, does the wire up to create a Chain
// component and performs a DB sanity check
func LaunchChain(eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus, counter *chainsync.Counter) (chain.Loader, error) {
	// creating and firing up the chain process
	genesis := cfg.DecodeGenesis()
	_, db := heavy.CreateDBConnection()
	l := chain.NewDBLoader(db, genesis)

	chainProcess, err := chain.New(eventBus, rpcBus, counter, l, l)
	if err != nil {
		return nil, err
	}

	// Perform database sanity check to ensure that it is rational before
	// bootstrapping all node subsystems
	if err := l.PerformSanityCheck(0, 10, 0); err != nil {
		return nil, err
	}

	go chainProcess.Listen()
	return l, nil
}

func (s *Server) launchKadcastPeer() {

	kcfg := cfg.Get().Kadcast

	if !kcfg.Enabled {
		log.Warn("Kadcast service is disabled")
		return
	}

	kadPeer := kadcast.NewPeer(s.eventBus, s.gossip, s.dupeMap)

	// Launch kadcast peer services and join network defined by bootstrappers
	kadPeer.Launch(kcfg.Address, kcfg.Bootstrappers, kcfg.MaxDelegatesNum)
	s.kadPeer = kadPeer
}

// Setup creates a new EventBus, generates the BLS and the ED25519 Keys,
// launches a new `CommitteeStore`, launches the Blockchain process, creates
// and launches a monitor client (if configuration demands it), and inits the
// Stake and Blind Bid channels
func Setup() *Server {
	// creating the eventbus
	eventBus := eventbus.New()

	counter := chainsync.NewCounter(eventBus)

	// creating the rpcbus
	rpcBus := rpcbus.New()

	m := mempool.NewMempool(eventBus, rpcBus, nil)
	m.Run()

	chainDBLoader, err := LaunchChain(eventBus, rpcBus, counter)

	if err != nil {
		log.Panic(err)
	}

	// Setting up the candidate broker
	candidateBroker := candidate.NewBroker(eventBus, rpcBus)
	go candidateBroker.Listen()

	// Setting up a dupemap
	dupeBlacklist := launchDupeMap(eventBus)

	// Instantiate gRPC server
	rpcWrapper, err := rpc.StartgRPCServer(rpcBus)
	if err != nil {
		log.WithError(err).Errorln("could not start gRPC server")
	}

	// Instantiate gRPC client
	// TODO: get address from config
	// client := rpc.InitRuskClient("127.0.0.1:8080", rpcBus)

	// Instantiate GraphQL server
	if cfg.Get().Gql.Enabled {
		if gqlServer, e := gql.NewHTTPServer(eventBus, rpcBus); e != nil {
			log.Errorf("GraphQL http server error: %v", e)
		} else {
			if e := gqlServer.Start(); e != nil {
				log.Errorf("GraphQL failed to start: %v", e)
			}
		}
	}

	// creating the Server
	srv := &Server{
		eventBus:   eventBus,
		rpcBus:     rpcBus,
		loader:     chainDBLoader,
		dupeMap:    dupeBlacklist,
		counter:    counter,
		gossip:     protocol.NewGossip(protocol.TestNet),
		rpcWrapper: rpcWrapper,
		// rpcClient:  client,
	}

	// Setting up the transactor component
	transactorComponent, err := transactor.New(eventBus, rpcBus, nil, srv.counter, nil, nil, cfg.Get().General.WalletOnly)
	if err != nil {
		log.Panic(err)
	}
	go transactorComponent.Listen()

	// Setting up and launch kadcast peer
	srv.launchKadcastPeer()

	// Connecting to the log based monitoring system
	stopFunc, err := LaunchMonitor(eventBus)
	if err != nil {
		log.Panic(err)
	}
	srv.cancelMonitor = stopFunc

	// if API is enabled, start it
	if cfg.Get().API.Address != "" {
		api.StartAPI()
	}
	return srv
}

func launchDupeMap(eventBus eventbus.Broker) *dupemap.DupeMap {
	acceptedBlockChan, _ := consensus.InitAcceptedBlockUpdate(eventBus)
	dupeBlacklist := dupemap.NewDupeMap(1)
	go func() {
		for {
			blk := <-acceptedBlockChan
			// NOTE: do we need locking?
			dupeBlacklist.UpdateHeight(blk.Header.Height)
		}
	}()
	return dupeBlacklist
}

// OnAccept read incoming packet from the peers
func (s *Server) OnAccept(conn net.Conn) {
	writeQueueChan := make(chan *bytes.Buffer, 1000)
	exitChan := make(chan struct{}, 1)
	peerReader, err := peer.NewReader(conn, s.gossip, s.dupeMap, s.eventBus, s.rpcBus, s.counter, writeQueueChan, exitChan)
	if err != nil {
		logServer.Panic(err)
	}

	if err := peerReader.Accept(); err != nil {
		logServer.WithError(err).Warnln("problem performing handshake")
		return
	}
	logServer.WithField("address", peerReader.Addr()).Debugln("connection established")

	go peerReader.ReadLoop()

	peerWriter := peer.NewWriter(conn, s.gossip, s.eventBus)
	go peerWriter.Serve(writeQueueChan, exitChan)
}

// OnConnection is the callback for writing to the peers
func (s *Server) OnConnection(conn net.Conn, addr string) {
	writeQueueChan := make(chan *bytes.Buffer, 1000)
	peerWriter := peer.NewWriter(conn, s.gossip, s.eventBus)

	if err := peerWriter.Connect(); err != nil {
		logServer.WithError(err).Warnln("problem performing handshake")
		return
	}
	logServer.WithField("address", peerWriter.Addr()).
		Debugln("connection established")

	exitChan := make(chan struct{}, 1)
	peerReader, err := peer.NewReader(conn, s.gossip, s.dupeMap, s.eventBus, s.rpcBus, s.counter, writeQueueChan, exitChan)
	if err != nil {
		log.Panic(err)
	}

	go peerReader.ReadLoop()
	go peerWriter.Serve(writeQueueChan, exitChan)
}

// Close the chain and the connections created through the RPC bus
func (s *Server) Close() {
	// TODO: disconnect peers
	_ = s.loader.Close(cfg.Get().Database.Driver)
	s.rpcBus.Close()
	s.rpcWrapper.Shutdown()
	// _ = s.rpcClient.Close()
	s.cancelMonitor()

	if s.kadPeer != nil {
		s.kadPeer.Close()
	}
}
