package main

import (
	"bytes"
	"net"

	"github.com/sirupsen/logrus"

	"github.com/dusk-network/dusk-blockchain/pkg/gql"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"

	"github.com/dusk-network/dusk-blockchain/pkg/core/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/chain"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/mempool"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactor"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
)

var logServer = logrus.WithField("process", "server")

// Server is the main process of the node
type Server struct {
	eventBus   *eventbus.EventBus
	rpcBus     *rpcbus.RPCBus
	chain      *chain.Chain
	dupeMap    *dupemap.DupeMap
	counter    *chainsync.Counter
	gossip     *processing.Gossip
	rpcWrapper *rpc.SrvWrapper
}

// Setup creates a new EventBus, generates the BLS and the ED25519 Keys, launches a new `CommitteeStore`, launches the Blockchain process and inits the Stake and Blind Bid channels
func Setup() *Server {
	// creating the eventbus
	eventBus := eventbus.New()

	counter := chainsync.NewCounter(eventBus)

	// creating the rpcbus
	rpcBus := rpcbus.New()

	m := mempool.NewMempool(eventBus, rpcBus, nil)
	m.Run()

	// creating and firing up the chain process
	chainProcess, err := chain.New(eventBus, rpcBus, counter)
	if err != nil {
		log.Panic(err)
	}
	go chainProcess.Listen()

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
		chain:      chainProcess,
		dupeMap:    dupeBlacklist,
		counter:    counter,
		gossip:     processing.NewGossip(protocol.TestNet),
		rpcWrapper: rpcWrapper,
	}

	// Setting up the transactor component
	transactorComponent, err := transactor.New(eventBus, rpcBus, nil, srv.counter, nil, nil, cfg.Get().General.WalletOnly)
	if err != nil {
		log.Panic(err)
	}
	go transactorComponent.Listen()

	// Connecting to the log based monitoring system
	if err := ConnectToLogMonitor(eventBus); err != nil {
		log.Panic(err)
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
	_ = s.chain.Close()
	s.rpcBus.Close()
	s.rpcWrapper.Shutdown()
}
