package main

import (
	"bytes"
	"context"
	"net"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/api"
	"github.com/dusk-network/dusk-blockchain/pkg/core/chain"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/dusk-network/dusk-blockchain/pkg/gql"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc/client"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc/server"
	"github.com/dusk-network/dusk-blockchain/pkg/util/legacy"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"

	"github.com/dusk-network/dusk-blockchain/pkg/core/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/chain2"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
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
	eventBus          *eventbus.EventBus
	rpcBus            *rpcbus.RPCBus
	loader            chain.Loader
	dupeMap           *dupemap.DupeMap
	counter           *chainsync.Counter
	gossip            *processing.Gossip
	grpcServer        *grpc.Server
	ruskConn          *grpc.ClientConn
	activeConnections map[string]time.Time
}

// LaunchChain instantiates a chain.Loader, does the wire up to create a Chain
// component and performs a DB sanity check
func LaunchChain(ctx context.Context, proxy transactions.Proxy, eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus, counter *chainsync.Counter, srv *grpc.Server, db database.DB) (chain.Loader, error) {
	// creating and firing up the chain process
	var genesis *block.Block
	if cfg.Get().Genesis.Legacy {
		g := legacy.DecodeGenesis()
		var err error
		genesis, err = legacy.OldBlockToNewBlock(g)
		if err != nil {
			return nil, err
		}
	} else {
		genesis = cfg.DecodeGenesis()
	}
	l := chain2.NewDBLoader(db, genesis)

	chainProcess, err := chain2.New(ctx, eventBus, rpcBus, proxy, l, proxy.Executor())
	if err != nil {
		return nil, err
	}

	// Perform database sanity check to ensure that it is rational before
	// bootstrapping all node subsystems
	if err := l.PerformSanityCheck(0, 10, 0); err != nil {
		return nil, err
	}

	go chainProcess.ConsensusLoop()
	return l, nil
}

// Setup creates a new EventBus, generates the BLS and the ED25519 Keys,
// launches a new `CommitteeStore`, launches the Blockchain process, creates
// and launches a monitor client (if configuration demands it), and inits the
// Stake and Blind Bid channels
func Setup() *Server {
	ctx := context.Background()
	grpcServer, err := server.SetupGRPC(server.FromCfg())
	if err != nil {
		log.Panic(err)
	}

	// creating the eventbus
	eventBus := eventbus.New()

	// creating the rpcbus
	rpcBus := rpcbus.New()

	counter := chainsync.NewCounter()

	// Instantiate gRPC client
	// TODO: get address from config
	ruskClient, ruskConn := client.CreateStateClient(ctx, cfg.Get().RPC.Rusk.Address)
	keysClient, _ := client.CreateKeysClient(ctx, cfg.Get().RPC.Rusk.Address)
	blindbidServiceClient, _ := client.CreateBlindBidServiceClient(ctx, cfg.Get().RPC.Rusk.Address)
	bidServiceClient, _ := client.CreateBidServiceClient(ctx, cfg.Get().RPC.Rusk.Address)
	transferClient, _ := client.CreateTransferClient(ctx, cfg.Get().RPC.Rusk.Address)
	stakeClient, _ := client.CreateStakeClient(ctx, cfg.Get().RPC.Rusk.Address)

	txTimeout := time.Duration(cfg.Get().RPC.Rusk.ContractTimeout) * time.Millisecond
	defaultTimeout := time.Duration(cfg.Get().RPC.Rusk.DefaultTimeout) * time.Millisecond
	proxy := transactions.NewProxy(ruskClient, keysClient, blindbidServiceClient, bidServiceClient, transferClient, stakeClient, txTimeout, defaultTimeout)

	m := mempool.NewMempool(ctx, eventBus, rpcBus, proxy.Prober(), grpcServer)
	m.Run()

	_, db := heavy.CreateDBConnection()
	// Instantiate API server
	if cfg.Get().API.Enabled {
		if apiServer, e := api.NewHTTPServer(eventBus, rpcBus); e != nil {
			log.Errorf("API http server error: %v", e)
		} else {
			go func() {
				if e := apiServer.Start(apiServer); e != nil {
					log.Errorf("API failed to start: %v", e)
				}
			}()
		}
	}

	chainDBLoader, err := LaunchChain(ctx, proxy, eventBus, rpcBus, counter, grpcServer, db)
	if err != nil {
		log.Panic(err)
	}

	// Setting up the candidate broker
	candidateBroker := candidate.NewBroker(eventBus, rpcBus)
	go candidateBroker.Listen()

	// Setting up a dupemap
	dupeBlacklist := launchDupeMap(eventBus)

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
		eventBus:          eventBus,
		rpcBus:            rpcBus,
		loader:            chainDBLoader,
		dupeMap:           dupeBlacklist,
		counter:           counter,
		gossip:            processing.NewGossip(protocol.TestNet),
		grpcServer:        grpcServer,
		ruskConn:          ruskConn,
		activeConnections: make(map[string]time.Time),
	}

	// Setting up the transactor component
	_, err = transactor.New(eventBus, rpcBus, nil, grpcServer, proxy)
	if err != nil {
		log.Panic(err)
	}

	// TODO: maintainer should be started here

	// Start serving from the gRPC server
	go func() {
		conf := cfg.Get().RPC
		l, err := net.Listen(conf.Network, conf.Address)
		if err != nil {
			log.Panic(err)
		}

		log.WithField("net", conf.Network).
			WithField("addr", conf.Address).Infof("gRPC HTTP server listening")

		if err := grpcServer.Serve(l); err != nil {
			log.WithError(err).Warn("Serve returned err")
		}
	}()

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
		panic(err)
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
	address := peerWriter.Addr()
	logServer.WithField("address", address).
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
	s.grpcServer.GracefulStop()
	_ = s.ruskConn.Close()
}
