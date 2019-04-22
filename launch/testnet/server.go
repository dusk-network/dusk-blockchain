package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"math"
	"math/big"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/bwesterb/go-ristretto"
	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/chain"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/factory"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/dupemap"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"gitlab.dusk.network/dusk-core/zkproof"
)

var timeOut = 3 * time.Second

type Server struct {
	eventBus *wire.EventBus

	// stakes are stored in a map, with the BLS public key as the key, so that they
	// can be properly removed when a node disconnects.
	sync.RWMutex
	stakes  map[string]*bytes.Buffer
	bids    []*bytes.Buffer
	chain   *chain.Chain
	dupeMap *dupemap.DupeMap

	// subscriber channels
	stakeChan             <-chan *bytes.Buffer
	bidChan               <-chan *bytes.Buffer
	removeProvisionerChan <-chan []byte
}

type txCollector struct {
	txChannel chan *bytes.Buffer
}

// Setup creates a new EventBus, generates the BLS and the ED25519 Keys, launches a new `CommitteeStore`, launches the Blockchain process and inits the Stake and Blind Bid channels
func Setup() *Server {
	// creating the eventbus
	eventBus := wire.NewEventBus()
	// generating the keys
	// TODO: this should probably lookup the keys on a local storage before recreating new ones
	keys, _ := user.NewRandKeys()

	// firing up the committee (the process in charge of ccalculating the quorum requirements and keeping track of the Provisioners eligible to vote according to the deterministic sortition)
	c := committee.LaunchCommitteeStore(eventBus, keys)

	// creating and firing up the chain process
	chain, err := chain.New(eventBus)
	if err != nil {
		panic(err)
	}
	go chain.Listen()

	// wiring up the Stake channel to the EventBus
	stakeChan := initStakeCollector(eventBus)
	// wiring up the BlindBid channel to the EventBus
	bidChan := initBidCollector(eventBus)

	dupeBlacklist := dupemap.NewDupeMap(eventBus)
	go dupeBlacklist.CleanOnRound()

	// creating the Server
	srv := &Server{
		eventBus:              eventBus,
		RWMutex:               sync.RWMutex{},
		stakes:                make(map[string]*bytes.Buffer),
		bids:                  []*bytes.Buffer{},
		chain:                 chain,
		stakeChan:             stakeChan,
		bidChan:               bidChan,
		removeProvisionerChan: committee.InitRemoveProvisionerCollector(eventBus),
		dupeMap:               dupeBlacklist,
	}

	// make a stake and bid tx
	stake := makeStake(keys)
	// bid is the blind bid, k is the secret to be embedded in the tx and d is the amount of Dusk locked in the blindbid. This is to be changed into the Commitment to d: D

	//NOTE: this is solely for testnet
	bid, d, k := makeBid()
	// saving the stake in the chain
	if err := chain.AcceptTx(stake); err != nil {
		panic(err)
	}

	// saving the bid in the chain
	if err := chain.AcceptTx(bid); err != nil {
		panic(err)
	}

	// Connecting to the general monitoring system
	ConnectToMonitor(eventBus, d)

	// start consensus factory
	factory := factory.New(eventBus, timeOut, c, keys, d, k)
	go factory.StartConsensus()

	return srv
}

func initStakeCollector(eventBus *wire.EventBus) chan *bytes.Buffer {
	stakeChannel := make(chan *bytes.Buffer, 100)
	collector := &txCollector{stakeChannel}
	go wire.NewTopicListener(eventBus, collector, msg.StakeTopic).Accept()
	return stakeChannel
}

func initBidCollector(eventBus *wire.EventBus) chan *bytes.Buffer {
	bidChannel := make(chan *bytes.Buffer, 100)
	collector := &txCollector{bidChannel}
	go wire.NewTopicListener(eventBus, collector, msg.BidTopic).Accept()
	return bidChannel
}

func (t *txCollector) Collect(r *bytes.Buffer) error {
	t.txChannel <- r
	return nil
}

// Listen to new Stake notification and Blind Bids
func (s *Server) Listen() {
	for {
		select {
		case b := <-s.stakeChan:
			log.WithField("process", "server").Debugln("stake received")
			buf, pubKeyBLS := extractBLSPubKey(b)
			s.Lock()
			s.stakes[pubKeyBLS] = buf
			s.Unlock()
		case b := <-s.bidChan:
			log.WithField("process", "server").Debugln("bid received")
			s.Lock()
			s.bids = append(s.bids, b)
			s.Unlock()
		case pubKeyBLS := <-s.removeProvisionerChan:
			pubKeyStr := hex.EncodeToString(pubKeyBLS)
			s.Lock()
			delete(s.stakes, pubKeyStr)
			s.Unlock()
		}
	}
}

// extract a provisioner's BLS public key from a staking tx buffer
func extractBLSPubKey(r *bytes.Buffer) (*bytes.Buffer, string) {
	buf := new(bytes.Buffer)
	reader := io.TeeReader(r, buf)
	stake := &transactions.Stake{}
	if err := stake.Decode(reader); err != nil {
		panic(err)
	}

	return buf, hex.EncodeToString(stake.PubKeyBLS)
}

func (s *Server) StartConsensus(round uint64) {
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes[:8], round)
	s.eventBus.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))
}

func (s *Server) OnAccept(conn net.Conn) {
	peer := peer.NewPeer(conn, true, protocol.TestNet, s.eventBus, s.dupeMap)
	// send the latest block height
	heightBytes := make([]byte, 8)
	s.chain.RLock()
	binary.LittleEndian.PutUint64(heightBytes, s.chain.PrevBlock.Header.Height)
	s.chain.RUnlock()
	if _, err := peer.Conn.Write(heightBytes); err != nil {
		log.WithFields(log.Fields{
			"process": "server",
			"error":   err,
		}).Warnln("problem writing to peer")
		peer.Disconnect()
		return
	}

	if err := peer.Run(); err != nil {
		log.WithFields(log.Fields{
			"process": "server",
			"error":   err,
		}).Warnln("problem performing handshake")
		return
	}
	log.WithFields(log.Fields{
		"process": "server",
		"address": peer.Conn.RemoteAddr().String(),
	}).Debugln("connection established")

	s.sendStakesAndBids(peer)
}

func (s *Server) OnConnection(conn net.Conn, addr string) {
	peer := peer.NewPeer(conn, false, protocol.TestNet, s.eventBus, s.dupeMap)
	// get latest block height
	buf := make([]byte, 8)
	if _, err := peer.Conn.Read(buf); err != nil {
		log.WithFields(log.Fields{
			"process": "server",
			"error":   err,
		}).Warnln("problem reading from peer")
		peer.Disconnect()
		return
	}

	height := binary.LittleEndian.Uint64(buf)
	s.chain.Lock()
	if s.chain.PrevBlock.Header.Height < height {
		s.chain.PrevBlock.Header.Height = height
	}
	s.chain.Unlock()

	if err := peer.Run(); err != nil {
		log.WithFields(log.Fields{
			"process": "server",
			"error":   err,
		}).Warnln("problem performing handshake")
		return
	}
	log.WithFields(log.Fields{
		"process": "server",
		"address": peer.Conn.RemoteAddr().String(),
	}).Debugln("connection established")

	s.sendStakesAndBids(peer)
}

func (s *Server) sendStakesAndBids(peer *peer.Peer) {
	s.RLock()
	defer s.RUnlock()
	for _, stake := range s.stakes {
		if err := peer.WriteMessage(stake, topics.Tx); err != nil {
			panic(err)
		}
	}

	for _, bid := range s.bids {
		if err := peer.WriteMessage(bid, topics.Tx); err != nil {
			panic(err)
		}
	}
}

func (s *Server) Close() {
	s.chain.Close()
}

func makeStake(keys *user.Keys) *transactions.Stake {
	stake, _ := transactions.NewStake(0, math.MaxUint64, 100, *keys.EdPubKey, keys.BLSPubKey.Marshal())
	keyImage, _ := crypto.RandEntropy(32)
	txID, _ := crypto.RandEntropy(32)
	signature, _ := crypto.RandEntropy(32)
	input, _ := transactions.NewInput(keyImage, txID, 0, signature)
	stake.Inputs = transactions.Inputs{input}

	outputAmount := rand.Int63n(100000)
	commitment := make([]byte, 32)
	binary.BigEndian.PutUint64(commitment[24:32], uint64(outputAmount))
	destKey, _ := crypto.RandEntropy(32)
	rangeProof, _ := crypto.RandEntropy(32)
	output, _ := transactions.NewOutput(commitment, destKey, rangeProof)
	stake.Outputs = transactions.Outputs{output}

	return stake
}

func makeBid() (*transactions.Bid, ristretto.Scalar, ristretto.Scalar) {
	k := ristretto.Scalar{}
	k.Rand()
	outputAmount := rand.Int63n(100000)
	d := big.NewInt(outputAmount)
	dScalar := ristretto.Scalar{}
	dScalar.SetBigInt(d)
	m := zkproof.CalculateM(k)
	bid, _ := transactions.NewBid(0, math.MaxUint64, 100, m.Bytes())
	keyImage, _ := crypto.RandEntropy(32)
	txID, _ := crypto.RandEntropy(32)
	signature, _ := crypto.RandEntropy(32)
	input, _ := transactions.NewInput(keyImage, txID, 0, signature)
	bid.Inputs = transactions.Inputs{input}

	commitment := make([]byte, 32)
	binary.BigEndian.PutUint64(commitment[24:32], uint64(outputAmount))
	destKey, _ := crypto.RandEntropy(32)
	rangeProof, _ := crypto.RandEntropy(32)
	output, _ := transactions.NewOutput(commitment, destKey, rangeProof)
	bid.Outputs = transactions.Outputs{output}

	return bid, dScalar, k
}
