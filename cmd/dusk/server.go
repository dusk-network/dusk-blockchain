package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/api"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/chain"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	walletdb "github.com/dusk-network/dusk-blockchain/pkg/core/data/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/core/loop"
	"github.com/dusk-network/dusk-blockchain/pkg/core/mempool"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactor"
	"github.com/dusk-network/dusk-blockchain/pkg/gql"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/kadcast"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc/client"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc/server"
	"github.com/dusk-network/dusk-blockchain/pkg/util/legacy"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
	"google.golang.org/grpc"
)

var logServer = logrus.WithField("process", "server")
var testnet = byte(2)

// Server is the main process of the node
type Server struct {
	eventBus          *eventbus.EventBus
	rpcBus            *rpcbus.RPCBus
	c                 *chain.Chain
	dupeMap           *dupemap.DupeMap
	gossip            *protocol.Gossip
	grpcServer        *grpc.Server
	ruskConn          *grpc.ClientConn
	readerFactory     *peer.ReaderFactory
	activeConnections map[string]time.Time
	kadPeer           *kadcast.Peer
}

// LaunchChain instantiates a chain.Loader, does the wire up to create a Chain
// component and performs a DB sanity check
func LaunchChain(ctx context.Context, cl *loop.Consensus, proxy transactions.Proxy, eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus, srv *grpc.Server, db database.DB) (*chain.Chain, error) {
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
	l := chain.NewDBLoader(db, genesis)

	chainProcess, err := chain.New(ctx, db, eventBus, rpcBus, l, l, srv, proxy, cl)
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

func (s *Server) launchKadcastPeer() {
	kcfg := cfg.Get().Kadcast

	if !kcfg.Enabled {
		log.Warn("Kadcast service is disabled")
		return
	}

	kadPeer := kadcast.NewPeer(s.eventBus, s.gossip, s.dupeMap, kcfg.Raptor)
	// Launch kadcast peer services and join network defined by bootstrappers
	kadPeer.Launch(kcfg.Address, kcfg.Bootstrappers, kcfg.MaxDelegatesNum)
	s.kadPeer = kadPeer
}

func getPassword() (string, error) {
	pw, err := readPassword("Enter password:")
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
// Stake and Blind Bid channels
func Setup() *Server {
	ctx := context.Background()
	var pw string
	if cfg.Get().General.TestHarness {
		pw = os.Getenv("DUSK_WALLET_PASS")
	} else {
		var err error
		pw, err = getPassword()
		if err != nil {
			log.Panic(err)
		}
	}

	grpcServer, err := server.SetupGRPC(server.FromCfg())
	if err != nil {
		log.Panic(err)
	}

	eventBus := eventbus.New()
	rpcBus := rpcbus.New()

	_, db := heavy.CreateDBConnection()

	processor := peer.NewMessageProcessor(eventBus)
	registerPeerServices(processor, db, eventBus, rpcBus)

	// Instantiate gRPC client
	// TODO: get address from config
	gctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Get().RPC.Rusk.ConnectionTimeout)*time.Millisecond)
	defer cancel()
	proxy, ruskConn := setupGRPCClients(gctx)

	var w *wallet.Wallet
	if _, err = os.Stat(cfg.Get().Wallet.File); err == nil {
		w, err = loadWallet(pw)
	} else {
		w, err = createWallet(nil, pw, proxy.KeyMaster())
	}
	if err != nil {
		log.Panic(err)
	}

	m := mempool.NewMempool(ctx, eventBus, rpcBus, proxy.Prober(), grpcServer)
	m.Run()
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

	e := &consensus.Emitter{
		EventBus:    eventBus,
		RPCBus:      rpcBus,
		Keys:        w.Keys(),
		Proxy:       proxy,
		TimerLength: cfg.ConsensusTimeOut,
	}

	cl := loop.New(e, &w.PublicKey)
	processor.Register(topics.Candidate, cl.ProcessCandidate)

	c, err := LaunchChain(ctx, cl, proxy, eventBus, rpcBus, grpcServer, db)
	if err != nil {
		log.Panic(err)
	}

	sync := chain.NewSynchronizer(ctx, eventBus, rpcBus, db, c)

	processor.Register(topics.Block, sync.ProcessBlock)

	dupeBlacklist := dupemap.Launch(eventBus)

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

	// Creating the peer factory
	readerFactory := peer.NewReaderFactory(processor)

	// creating the Server
	srv := &Server{
		eventBus:          eventBus,
		rpcBus:            rpcBus,
		c:                 c,
		dupeMap:           dupeBlacklist,
		gossip:            protocol.NewGossip(protocol.TestNet),
		grpcServer:        grpcServer,
		ruskConn:          ruskConn,
		readerFactory:     readerFactory,
		activeConnections: make(map[string]time.Time),
	}

	// Setting up the transactor component
	_, err = transactor.New(eventBus, rpcBus, nil, grpcServer, proxy, w)
	if err != nil {
		log.Panic(err)
	}

	// Setting up and launch kadcast peer
	srv.launchKadcastPeer()

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

	go func() {
		if err := c.ProduceBlock(); err != nil {
			log.WithError(err).Warn("ProduceBlock returned err")
		}
	}()

	return srv
}

// OnAccept read incoming packet from the peers
func (s *Server) OnAccept(conn net.Conn) {
	writeQueueChan := make(chan bytes.Buffer, 1000)
	peerReader := s.readerFactory.SpawnReader(conn, s.gossip, s.dupeMap, writeQueueChan)
	if err := peerReader.Accept(); err != nil {
		logServer.WithError(err).Warnln("OnAccept, problem performing handshake")
		return
	}
	logServer.WithField("address", peerReader.Addr()).Debugln("connection established")

	peerWriter := peer.NewWriter(conn, s.gossip, s.eventBus)
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		err := peer.Create(ctx, peerReader, peerWriter, writeQueueChan)
		logServer.WithError(err).Warnln("error received from peer")
	}()
}

// OnConnection is the callback for writing to the peers
func (s *Server) OnConnection(conn net.Conn, addr string) {
	writeQueueChan := make(chan bytes.Buffer, 1000)
	peerWriter := peer.NewWriter(conn, s.gossip, s.eventBus)

	if err := peerWriter.Connect(); err != nil {
		logServer.WithError(err).Warnln("OnConnection, problem performing handshake")
		return
	}
	address := peerWriter.Addr()
	logServer.WithField("address", address).
		Debugln("connection established")

	peerReader := s.readerFactory.SpawnReader(conn, s.gossip, s.dupeMap, writeQueueChan)

	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		err := peer.Create(ctx, peerReader, peerWriter, writeQueueChan)
		logServer.WithError(err).Warnln("error received from peer")
	}()
}

// Close the chain and the connections created through the RPC bus
func (s *Server) Close() {
	// TODO: disconnect peers
	// _ = s.c.Close(cfg.Get().Database.Driver)
	s.rpcBus.Close()
	s.grpcServer.GracefulStop()
	_ = s.ruskConn.Close()

	if s.kadPeer != nil {
		s.kadPeer.Close()
	}
}

func registerPeerServices(processor *peer.MessageProcessor, db database.DB, eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus) {
	processor.Register(topics.Ping, responding.ProcessPing)
	dataBroker := responding.NewDataBroker(db, rpcBus)
	processor.Register(topics.GetData, dataBroker.SendItems)
	dataRequestor := responding.NewDataRequestor(db, rpcBus, eventBus)
	processor.Register(topics.Inv, dataRequestor.RequestMissingItems)
	bhb := responding.NewBlockHashBroker(db)
	processor.Register(topics.GetBlocks, bhb.AdvertiseMissingBlocks)
	cb := responding.NewCandidateBroker(db)
	processor.Register(topics.GetCandidate, cb.ProvideCandidate)
	cp := consensus.NewPublisher(eventBus)
	processor.Register(topics.Score, cp.Process)
	processor.Register(topics.Reduction, cp.Process)
	processor.Register(topics.Agreement, cp.Process)
}

func setupGRPCClients(ctx context.Context) (transactions.Proxy, *grpc.ClientConn) {
	ruskClient, ruskConn := client.CreateStateClient(ctx, cfg.Get().RPC.Rusk.Address)
	keysClient, _ := client.CreateKeysClient(ctx, cfg.Get().RPC.Rusk.Address)
	blindbidServiceClient, _ := client.CreateBlindBidServiceClient(ctx, cfg.Get().RPC.Rusk.Address)
	bidServiceClient, _ := client.CreateBidServiceClient(ctx, cfg.Get().RPC.Rusk.Address)
	transferClient, _ := client.CreateTransferClient(ctx, cfg.Get().RPC.Rusk.Address)
	stakeClient, _ := client.CreateStakeClient(ctx, cfg.Get().RPC.Rusk.Address)

	txTimeout := time.Duration(cfg.Get().RPC.Rusk.ContractTimeout) * time.Millisecond
	defaultTimeout := time.Duration(cfg.Get().RPC.Rusk.DefaultTimeout) * time.Millisecond
	return transactions.NewProxy(ruskClient, keysClient, blindbidServiceClient, bidServiceClient, transferClient, stakeClient, txTimeout, defaultTimeout), ruskConn
}

func loadWallet(password string) (*wallet.Wallet, error) {
	// First load the database
	db, err := walletdb.New(cfg.Get().Wallet.Store)
	if err != nil {
		return nil, err
	}

	// Then load the wallet
	return wallet.LoadFromFile(testnet, db, password, cfg.Get().Wallet.File)
}

func createWallet(seed []byte, password string, keyMaster transactions.KeyMaster) (*wallet.Wallet, error) {
	// First load the database
	db, err := walletdb.New(cfg.Get().Wallet.Store)
	if err != nil {
		return nil, err
	}

	if seed == nil {
		seed, err = wallet.GenerateNewSeed(nil)
		if err != nil {
			return nil, err
		}
	}

	sk, pk, vk, err := keyMaster.GenerateKeys(context.Background(), seed)
	if err != nil {
		return nil, err
	}

	skBuf := new(bytes.Buffer)
	if err = keys.MarshalSecretKey(skBuf, &sk); err != nil {
		_ = db.Close()
		return nil, err
	}

	keysJSON := wallet.KeysJSON{
		Seed:      seed,
		SecretKey: skBuf.Bytes(),
		PublicKey: pk,
		ViewKey:   vk,
	}

	// Then create the wallet with seed and password
	w, err := wallet.LoadFromSeed(testnet, db, password, cfg.Get().Wallet.File, keysJSON)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	return w, nil
}
