package main

import (
	"bytes"
	"net"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/chain"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/mempool"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/dupemap"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing/chainsync"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/rpc"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
)

// Server is the main process of the node
type Server struct {
	eventBus *wire.EventBus
	rpcBus   *wire.RPCBus
	chain    *chain.Chain
	dupeMap  *dupemap.DupeMap
	counter  *chainsync.Counter
}

// Setup creates a new EventBus, generates the BLS and the ED25519 Keys, launches a new `CommitteeStore`, launches the Blockchain process and inits the Stake and Blind Bid channels
func Setup() *Server {
	// creating the eventbus
	eventBus := wire.NewEventBus()

	// creating the rpcbus
	rpcBus := wire.NewRPCBus()

	m := mempool.NewMempool(eventBus, nil)
	m.Run()

	// creating and firing up the chain process
	chain, err := chain.New(eventBus, rpcBus, nil)
	if err != nil {
		panic(err)
	}
	go chain.Listen()

	// Setting up a dupemap
	dupeBlacklist := launchDupeMap(eventBus)

	if cfg.Get().RPC.Enabled {
		rpcServ, err := rpc.NewRPCServer(eventBus, rpcBus)
		if err != nil {
			log.Errorf("RPC server error: %s", err.Error())
		}

		if err := rpcServ.Start(); err != nil {
			log.Errorf("RPC server error: %s", err.Error())
		}
	}

	// creating the Server
	srv := &Server{
		eventBus: eventBus,
		rpcBus:   rpcBus,
		chain:    chain,
		dupeMap:  dupeBlacklist,
		counter:  chainsync.NewCounter(eventBus),
	}

	// Connecting to the log based monitoring system
	if err := ConnectToLogMonitor(eventBus); err != nil {
		panic(err)
	}

	gossip := processing.NewGossip(protocol.TestNet)
	eventBus.RegisterPreprocessor(string(topics.Gossip), gossip)

	return srv
}

func launchDupeMap(eventBus wire.EventBroker) *dupemap.DupeMap {
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
	peerReader, err := peer.NewReader(conn, protocol.TestNet, s.dupeMap, s.eventBus, s.rpcBus, s.counter, writeQueueChan, exitChan)
	if err != nil {
		panic(err)
	}

	if err := peerReader.Accept(); err != nil {
		log.WithFields(log.Fields{
			"process": "server",
			"error":   err,
		}).Warnln("problem performing handshake")
		return
	}
	log.WithFields(log.Fields{
		"process": "server",
		"address": peerReader.Addr(),
	}).Debugln("connection established")

	go peerReader.ReadLoop()

	peerWriter := peer.NewWriter(conn, protocol.TestNet, s.eventBus)
	go peerWriter.Serve(writeQueueChan, exitChan)
}

// OnConnection is the callback for writing to the peers
func (s *Server) OnConnection(conn net.Conn, addr string) {
	writeQueueChan := make(chan *bytes.Buffer, 1000)
	peerWriter := peer.NewWriter(conn, protocol.TestNet, s.eventBus)

	if err := peerWriter.Connect(); err != nil {
		log.WithFields(log.Fields{
			"process": "server",
			"error":   err,
		}).Warnln("problem performing handshake")
		return
	}
	log.WithFields(log.Fields{
		"process": "server",
		"address": peerWriter.Addr(),
	}).Debugln("connection established")

	exitChan := make(chan struct{}, 1)
	peerReader, err := peer.NewReader(conn, protocol.TestNet, s.dupeMap, s.eventBus, s.rpcBus, s.counter, writeQueueChan, exitChan)
	if err != nil {
		panic(err)
	}

	go peerReader.ReadLoop()
	go peerWriter.Serve(writeQueueChan, exitChan)
}

// Close the chain and the connections created through the RPC bus
func (s *Server) Close() {
	// TODO: disconnect peers
	s.chain.Close()
	s.rpcBus.Close()
}
