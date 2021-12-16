// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/api"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/config/genesis"
	"github.com/dusk-network/dusk-blockchain/pkg/core/chain"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	consensuskey "github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/core/loop"
	"github.com/dusk-network/dusk-blockchain/pkg/core/mempool"
	"github.com/dusk-network/dusk-blockchain/pkg/gql"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc/client"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc/server"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"golang.org/x/crypto/ssh/terminal"
	"google.golang.org/grpc"
)

var testnet = byte(2)

const voucherRetryTime = 15 * time.Second

// Server is the main process of the node.
type Server struct {
	eventBus *eventbus.EventBus
	rpcBus   *rpcbus.RPCBus
	c        *chain.Chain
	gossip   *protocol.Gossip

	grpcServer *grpc.Server
	gqlServer  *gql.Server

	ruskConn      *grpc.ClientConn
	readerFactory *peer.ReaderFactory
	kadPeer       *kadcast.Peer

	dbDriver database.Driver

	// Parent context to all long-lived goroutines triggered by any subsystem.
	ctx    context.Context
	cancel context.CancelFunc
}

// LaunchChain instantiates a chain.Loader, does the wire up to create a Chain
// component and performs a DB sanity check.
func LaunchChain(ctx context.Context, cl *loop.Consensus, proxy transactions.Proxy, eventBus *eventbus.EventBus, rpcbus *rpcbus.RPCBus, srv *grpc.Server, db database.DB) (*chain.Chain, error) {
	// creating and firing up the chain process
	genesis := genesis.Decode()
	l := chain.NewDBLoader(db, genesis)

	chainProcess, err := chain.New(ctx, db, eventBus, rpcbus, l, l, srv, proxy, cl)
	if err != nil {
		return nil, err
	}

	// Perform database sanity check to ensure that it is rational before
	// bootstrapping all node subsystems
	if err := l.PerformSanityCheck(0, 10, 0); err != nil {
		return nil, err
	}

	return chainProcess, nil
}

func (s *Server) launchKadcastPeer(ctx context.Context, p *peer.MessageProcessor, g *protocol.Gossip) {
	// launch kadcast client
	kadPeer := kadcast.NewKadcastPeer(ctx, s.eventBus, p, g)
	kadPeer.Launch()
	s.kadPeer = kadPeer
}

func getPassword(prompt string) (string, error) {
	pw, err := readPassword(prompt)
	return string(pw), err
}

// This is to bypass issue with stdin from non-tty.
func readPassword(prompt string) ([]byte, error) {
	fd := int(os.Stdin.Fd())
	if terminal.IsTerminal(fd) {
		fmt.Fprintln(os.Stderr, prompt)
		return terminal.ReadPassword(fd)
	}

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return scanner.Bytes(), nil
	}

	return nil, scanner.Err()
}

// Setup creates a new EventBus, generates the BLS and the ED25519 Keys,
// launches a new `CommitteeStore`, launches the Blockchain process, creates
// and launches a monitor client (if configuration demands it), and inits the
// Stake and Blind Bid channels.
func Setup() *Server {
	parentCtx, parentCancel := context.WithCancel(context.Background())
	_, err := os.Stat(cfg.Get().Wallet.File)

	grpcServer, err := server.SetupGRPC(server.FromCfg())
	if err != nil {
		log.Panic(err)
	}

	_ = newConfigService(grpcServer)

	eventBus := eventbus.New()
	rpcBus := rpcbus.New()

	driver, db := heavy.CreateDBConnection()

	processor := peer.NewMessageProcessor(eventBus)
	registerPeerServices(processor, db, eventBus, rpcBus)

	// Instantiate gRPC client
	// TODO: get address from config
	gctx, cancel := context.WithTimeout(parentCtx, time.Duration(cfg.Get().RPC.Rusk.ConnectionTimeout)*time.Millisecond)
	defer cancel()

	proxy, ruskConn := setupGRPCClients(gctx)

	m := mempool.NewMempool(db, eventBus, rpcBus, proxy.Prober(), grpcServer)
	m.Run(parentCtx)

	processor.Register(topics.Tx, m.ProcessTx)

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

	// TODO: consensus keys
	e := &consensus.Emitter{
		EventBus:    eventBus,
		RPCBus:      rpcBus,
		Keys:        consensuskey.Keys{},
		TimerLength: cfg.ConsensusTimeOut,
	}

	// TODO: Wallet Public here needed?
	cl := loop.New(e, nil)
	processor.Register(topics.Candidate, cl.ProcessCandidate)

	c, err := LaunchChain(parentCtx, cl, proxy, eventBus, rpcBus, grpcServer, db)
	if err != nil {
		log.Panic(err)
	}

	processor.Register(topics.Block, c.ProcessBlockFromNetwork)

	// Instantiate GraphQL server
	var gqlServer *gql.Server

	if cfg.Get().Gql.Enabled {
		var e error
		if gqlServer, e = gql.NewHTTPServer(eventBus, rpcBus); e != nil {
			log.WithError(e).Error("graphq server failed to run")
		} else {
			if e = gqlServer.Start(parentCtx); e != nil {
				log.WithError(e).Error("graphq server failed to run")
			}
		}
	}

	// Creating the peer factory
	readerFactory := peer.NewReaderFactory(processor)

	// Create the listener and contact the voucher seeder
	gossip := protocol.NewGossip(protocol.TestNet)

	if !cfg.Get().Kadcast.Enabled {
		connector := peer.NewConnector(eventBus, gossip, cfg.Get().Network.Port, processor, protocol.ServiceFlag(cfg.Get().Network.ServiceFlag), peer.Create)

		seeders := cfg.Get().Network.Seeder.Addresses
		if err = connectToSeeders(connector, seeders); err != nil {
			panic("could not contact any voucher seeders")
		}
	}

	// creating the Server
	srv := &Server{
		eventBus:      eventBus,
		rpcBus:        rpcBus,
		c:             c,
		gossip:        gossip,
		gqlServer:     gqlServer,
		grpcServer:    grpcServer,
		ruskConn:      ruskConn,
		readerFactory: readerFactory,
		dbDriver:      driver,
		ctx:           parentCtx,
		cancel:        parentCancel,
	}

	// Setting up and launch kadcast peer
	kcfg := cfg.Get().Kadcast
	if kcfg.Enabled {
		srv.launchKadcastPeer(parentCtx, processor, gossip)
	}

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

	if err := c.RestartConsensus(); err != nil {
		log.WithError(err).Warn("StartConsensus returned err")
		// If we can not start consensus, we shouldn't be able to start at all.
		panic(err)
	}

	return srv
}

// Close the chain and the connections created through the RPC bus.
func (s *Server) Close() {
	// Cancel all goroutines long-lived loops
	s.cancel()

	// Close graphql server.
	if s.gqlServer != nil {
		if err := s.gqlServer.Close(); err != nil {
			log.WithError(err).Warn("failed to close gql server")
		}
	}

	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}

	// Close Rusk client connection
	_ = s.ruskConn.Close()

	// kadcast client grpc
	if s.kadPeer != nil {
		s.kadPeer.Close()
	}

	if s.dbDriver != nil {
		if err := s.dbDriver.Close(); err != nil {
			log.WithError(err).Warn("failed to close db driver")
		}
	}

	s.rpcBus.Close()
	s.eventBus.Close()
}

func connectToSeeders(connector *peer.Connector, seeders []string) error {
	i := 0
	var connected bool

	for {
		for _, seeder := range seeders {
			if err := connector.Connect(seeder); err != nil {
				log.WithError(err).Error("could not contact voucher seeder")
			} else {
				connected = true
			}
		}

		if connected {
			break
		}

		i++
		if i >= 3 {
			return errors.New("could not connect to any voucher seeders")
		}

		time.Sleep(voucherRetryTime * time.Duration(i))
	}

	return nil
}

func registerPeerServices(processor *peer.MessageProcessor, db database.DB, eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus) {
	processor.Register(topics.Ping, responding.ProcessPing)
	dataBroker := responding.NewDataBroker(db, rpcBus)
	dataRequestor := responding.NewDataRequestor(db, rpcBus)
	bhb := responding.NewBlockHashBroker(db)
	cb := responding.NewCandidateBroker(db)
	cp := consensus.NewPublisher(eventBus)

	processor.Register(topics.GetData, dataBroker.MarshalObjects)
	processor.Register(topics.MemPool, dataBroker.MarshalMempoolTxs)
	processor.Register(topics.Ping, responding.ProcessPing)
	processor.Register(topics.Pong, responding.ProcessPong)
	processor.Register(topics.Inv, dataRequestor.RequestMissingItems)
	processor.Register(topics.GetBlocks, bhb.AdvertiseMissingBlocks)
	processor.Register(topics.GetCandidate, cb.ProvideCandidate)
	processor.Register(topics.NewBlock, cp.Process)
	processor.Register(topics.Reduction, cp.Process)
	processor.Register(topics.Agreement, cp.Process)
	processor.Register(topics.AggrAgreement, cp.Process)
	processor.Register(topics.Challenge, responding.CompleteChallenge)
}

func setupGRPCClients(ctx context.Context) (transactions.Proxy, *grpc.ClientConn) {
	addr := cfg.Get().RPC.Rusk.Address
	if cfg.Get().RPC.Rusk.Network == "unix" {
		addr = "unix://" + cfg.Get().RPC.Rusk.Address
	}

	ruskClient, ruskConn := client.CreateStateClient(ctx, addr)
	keysClient, _ := client.CreateKeysClient(ctx, addr)
	transferClient, _ := client.CreateTransferClient(ctx, addr)
	stakeClient, _ := client.CreateStakeClient(ctx, addr)

	txTimeout := time.Duration(cfg.Get().RPC.Rusk.ContractTimeout) * time.Millisecond
	defaultTimeout := time.Duration(cfg.Get().RPC.Rusk.DefaultTimeout) * time.Millisecond
	return transactions.NewProxy(ruskClient, keysClient, transferClient, stakeClient, txTimeout, defaultTimeout), ruskConn
}
