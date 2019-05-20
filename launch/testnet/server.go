package main

import (
	"bytes"
	"encoding/binary"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/chain"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/mempool"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/dupemap"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/rpc"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
)

var timeOut = 3 * time.Second

type Server struct {
	eventBus  *wire.EventBus
	rpcBus    *wire.RPCBus
	chain     *chain.Chain
	collector *peer.MessageCollector
}

// Setup creates a new EventBus, generates the BLS and the ED25519 Keys, launches a new `CommitteeStore`, launches the Blockchain process and inits the Stake and Blind Bid channels
func Setup() *Server {
	// creating the eventbus
	eventBus := wire.NewEventBus()

	// creating the rpcbus
	rpcBus := wire.NewRPCBus()

	// generating the keys
	// TODO: this should probably lookup the keys on a local storage before recreating new ones
	keys, _ := user.NewRandKeys()

	// firing up the committee (the process in charge of ccalculating the quorum requirements and keeping track of the Provisioners eligible to vote according to the deterministic sortition)
	_ = committee.LaunchCommitteeStore(eventBus, keys)

	m := mempool.NewMempool(eventBus, nil)
	m.Run()

	// creating and firing up the chain process
	chain, err := chain.New(eventBus, rpcBus)
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
		collector: &peer.MessageCollector{
			Publisher:     eventBus,
			DupeBlacklist: dupeBlacklist,
			Magic:         protocol.TestNet,
		},
	}

	// make a stake and bid tx
	// stake := makeStake(keys)
	// bid is the blind bid, k is the secret to be embedded in the tx and d is the amount of Dusk locked in the blindbid. This is to be changed into the Commitment to d: D

	//NOTE: this is solely for testnet
	// bid, d, k := makeBid()

	// TODO: bid and stake creation should be handled by using the wallet, and not
	// directly put in the chain, but rather broadcasted.
	// Publish the stake in the chain
	// buf := new(bytes.Buffer)
	// err = stake.Encode(buf)
	// if err != nil {
	// 	log.Error(err)
	// }
	// eventBus.Publish(string(topics.Tx), buf)
	// Publish the bid in the chain
	// buf = new(bytes.Buffer)
	// err = bid.Encode(buf)
	// if err != nil {
	// 	log.Error(err)
	// }
	// eventBus.Publish(string(topics.Tx), buf)

	// Connecting to the general monitoring system
	// ConnectToMonitor(eventBus, d)

	// TODO: need to get the d and k here from the previously created bid tx
	// start consensus factory
	// factory := factory.New(eventBus, timeOut, c, keys, d, k)
	// go factory.StartConsensus()

	return srv
}

func launchDupeMap(eventBus wire.EventBroker) *dupemap.DupeMap {
	roundChan := consensus.InitRoundUpdate(eventBus)
	dupeBlacklist := dupemap.NewDupeMap(1)
	go func() {
		for {
			round := <-roundChan
			// NOTE: do we need locking?
			dupeBlacklist.UpdateHeight(round)
		}
	}()
	return dupeBlacklist
}

func (s *Server) StartConsensus(round uint64) {
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes[:8], round)
	s.eventBus.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))
}

func (s *Server) OnAccept(conn net.Conn) {
	peerReader := peer.NewReader(conn, protocol.TestNet)

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

	go peerReader.ReadLoop(s.collector)
	peerWriter := peer.NewWriter(conn, protocol.TestNet, s.eventBus)
	peerWriter.Subscribe()
}

func (s *Server) OnConnection(conn net.Conn, addr string) {
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

	peerReader := peer.NewReader(conn, protocol.TestNet)
	go peerReader.ReadLoop(s.collector)
}

func (s *Server) Close() {
	s.chain.Close()
	s.rpcBus.Close()
}
