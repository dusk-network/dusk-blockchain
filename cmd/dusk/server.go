// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/api"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/chain"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	consensuskey "github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/stakeautomaton"
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
	eventBus      *eventbus.EventBus
	rpcBus        *rpcbus.RPCBus
	c             *chain.Chain
	gossip        *protocol.Gossip
	grpcServer    *grpc.Server
	ruskConn      *grpc.ClientConn
	readerFactory *peer.ReaderFactory
	kadPeer       *kadcast.Peer
}

// LaunchChain instantiates a chain.Loader, does the wire up to create a Chain
// component and performs a DB sanity check.
func LaunchChain(ctx context.Context, cl *loop.Consensus, proxy transactions.Proxy, eventBus *eventbus.EventBus, srv *grpc.Server, db database.DB) (*chain.Chain, error) {
	// creating and firing up the chain process
	genesis := cfg.DecodeGenesis()
	l := chain.NewDBLoader(db, genesis)

	chainProcess, err := chain.New(ctx, db, eventBus, l, l, srv, proxy, cl)
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

func (s *Server) launchKadcastPeer(p *peer.MessageProcessor) {
	kcfg := cfg.Get().Kadcast

	if !kcfg.Enabled {
		log.Warn("Kadcast service is disabled")
		return
	}

	kadPeer := kadcast.NewPeer(s.eventBus, s.gossip, nil, p, kcfg.Raptor)
	// Launch kadcast peer services and join network defined by bootstrappers
	kadPeer.Launch(kcfg.Address, kcfg.Bootstrappers, kcfg.MaxDelegatesNum)
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
	var pw string

	ctx := context.Background()
	_, err := os.Stat(cfg.Get().Wallet.File)

	switch {
	case cfg.Get().General.TestHarness:
		pw = os.Getenv("DUSK_WALLET_PASS")
	case os.IsNotExist(err):
		fmt.Fprintln(os.Stderr, "Wallet file not found. Creating new file...")

		for {
			pw, err = getPassword("Enter password:")
			if err != nil {
				log.Panic(err)
			}

			var pw2 string

			pw2, err = getPassword("Confirm password:")
			if err != nil {
				log.Panic(err)
			}

			if pw == pw2 {
				break
			}
		}
	case err != nil:
		log.Panic(err)
	default:
		pw, err = getPassword("Enter password:")
		if err != nil {
			log.Panic(err)
		}
	}

	grpcServer, err := server.SetupGRPC(server.FromCfg())
	if err != nil {
		log.Panic(err)
	}

	_ = newConfigService(grpcServer)

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

	m := mempool.NewMempool(eventBus, rpcBus, proxy.Prober(), grpcServer)
	m.Run(ctx)
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
		TimerLength: cfg.ConsensusTimeOut,
	}

	cl := loop.New(e, &w.PublicKey)
	processor.Register(topics.Candidate, cl.ProcessCandidate)

	c, err := LaunchChain(ctx, cl, proxy, eventBus, grpcServer, db)
	if err != nil {
		log.Panic(err)
	}

	processor.Register(topics.Block, c.ProcessBlockFromNetwork)

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
		grpcServer:    grpcServer,
		ruskConn:      ruskConn,
		readerFactory: readerFactory,
	}

	// Setting up the transactor component
	_, err = transactor.New(eventBus, rpcBus, nil, grpcServer, proxy, w, c.CalculateSyncProgress)
	if err != nil {
		log.Panic(err)
	}

	_ = stakeautomaton.New(eventBus, rpcBus, grpcServer)

	// Setting up and launch kadcast peer
	srv.launchKadcastPeer(processor)

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

	go c.StartConsensus()

	return srv
}

// Close the chain and the connections created through the RPC bus.
func (s *Server) Close() {
	// TODO: disconnect peers
	// _ = s.c.Close(cfg.Get().Database.Driver)
	s.c.StopConsensus()
	s.rpcBus.Close()
	s.grpcServer.GracefulStop()
	_ = s.ruskConn.Close()

	if s.kadPeer != nil {
		s.kadPeer.Close()
	}
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
	walletClient, _ := client.CreateWalletClient(ctx, addr)

	txTimeout := time.Duration(cfg.Get().RPC.Rusk.ContractTimeout) * time.Millisecond
	defaultTimeout := time.Duration(cfg.Get().RPC.Rusk.DefaultTimeout) * time.Millisecond
	return transactions.NewProxy(ruskClient, keysClient, transferClient, stakeClient, walletClient, txTimeout, defaultTimeout), ruskConn
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

	consensusKeys := consensuskey.NewRandKeys()

	keysJSON.SecretKeyBLS = consensusKeys.BLSSecretKey
	keysJSON.PublicKeyBLS = consensusKeys.BLSPubKey

	// Then create the wallet with seed and password
	w, err := wallet.LoadFromSeed(testnet, db, password, cfg.Get().Wallet.File, keysJSON)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	return w, nil
}
