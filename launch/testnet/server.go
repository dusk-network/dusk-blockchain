package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/big"
	"math/rand"
	"net"
	"time"

	"github.com/bwesterb/go-ristretto"
	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/chain"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/factory"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"gitlab.dusk.network/dusk-core/zkproof"
)

var timeOut = 6 * time.Second

type Server struct {
	eventBus *wire.EventBus
	stakes   []*bytes.Buffer
	bids     []*bytes.Buffer
	chain    *chain.Chain

	// subscriber channels
	stakeChannel chan *bytes.Buffer
	bidChannel   chan *bytes.Buffer
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
	committee := committee.NewCommitteeStore(eventBus)
	go committee.Listen()

	// creating and firing the chain process
	chain, err := chain.New(eventBus)
	if err != nil {
		panic(err)
	}
	go chain.Listen()

	// wiring up the Stake channel to the EventBus
	stakeChannel := initStakeCollector(eventBus)
	// wiring up the BlinBid channel to the EventBus
	bidChannel := initBidCollector(eventBus)

	// creating the Server
	srv := &Server{
		eventBus:     eventBus,
		stakes:       []*bytes.Buffer{},
		bids:         []*bytes.Buffer{},
		chain:        chain,
		stakeChannel: stakeChannel,
		bidChannel:   bidChannel,
	}

	// make a stake and bid tx
	stake := makeStake(keys)
	// bid is the blind bid, k is the secret to be embedded in the tx and d is the amount of Dusk locked in the blindbid. This is to be changed into the Commitment to d: D

	//NOTE: this is solely for testnet
	bid, d, k := makeBid()
	// Publish the stake in the chain
	buf := new(bytes.Buffer)
	err = stake.Encode(buf)
	if err != nil {
		log.Error(err)
	}
	eventBus.Publish(string(topics.Tx), buf)
	// Publish the bid in the chain
	buf = new(bytes.Buffer)
	err = bid.Encode(buf)
	if err != nil {
		log.Error(err)
	}
	eventBus.Publish(string(topics.Tx), buf)

	// start consensus factory
	factory := factory.New(eventBus, timeOut, committee, chain.BidList, keys, d, k)
	go factory.StartConsensus()

	return srv
}

func initStakeCollector(eventBus *wire.EventBus) chan *bytes.Buffer {
	stakeChannel := make(chan *bytes.Buffer, 100)
	collector := &txCollector{stakeChannel}
	go wire.NewEventSubscriber(eventBus, collector, msg.StakeTopic).Accept()
	return stakeChannel
}

func initBidCollector(eventBus *wire.EventBus) chan *bytes.Buffer {
	bidChannel := make(chan *bytes.Buffer, 100)
	collector := &txCollector{bidChannel}
	go wire.NewEventSubscriber(eventBus, collector, msg.BidTopic).Accept()
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
		case b := <-s.stakeChannel:
			log.WithField("process", "server").Debugln("stake received")
			s.stakes = append(s.stakes, b)
		case b := <-s.bidChannel:
			log.WithField("process", "server").Debugln("bid received")
			s.bids = append(s.bids, b)
		}
	}
}

func (s *Server) StartConsensus(round uint64) {
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes[:8], round)
	s.eventBus.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))
}

func (s *Server) OnAccept(conn net.Conn) {
	log.WithFields(log.Fields{
		"process": "server",
		"address": conn.RemoteAddr().String(),
	}).Debugln("attempting to accept a connection")

	peer := peer.NewPeer(conn, true, protocol.TestNet, s.eventBus)
	// send the latest block
	buffer := new(bytes.Buffer)
	if err := s.chain.PrevBlock.Encode(buffer); err != nil {
		log.WithFields(log.Fields{
			"process": "server",
			"error":   err,
		}).Warnln("problem encoding previous block")
		peer.Disconnect()
		return
	}

	if _, err := peer.Conn.Write(buffer.Bytes()); err != nil {
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
	log.WithFields(log.Fields{
		"process": "server",
		"address": conn.RemoteAddr().String(),
	}).Debugln("attempting to make a connection")
	peer := peer.NewPeer(conn, false, protocol.TestNet, s.eventBus)
	// get latest block
	buf := make([]byte, 1024)
	if _, err := peer.Conn.Read(buf); err != nil {
		log.WithFields(log.Fields{
			"process": "server",
			"error":   err,
		}).Warnln("problem reading from peer")
		peer.Disconnect()
		return
	}
	var blk block.Block
	if err := blk.Decode(bytes.NewBuffer(buf)); err != nil {
		log.WithFields(log.Fields{
			"process": "server",
			"error":   err,
		}).Warnln("problem decoding block")
		peer.Disconnect()
		return
	}

	if s.chain.PrevBlock.Header.Height < blk.Header.Height {
		s.chain.PrevBlock = blk
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

func (s *Server) sendStakesAndBids(peer *peer.Peer) {
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

func makeProvisionerBytes(keys *user.Keys, amount uint64) (*bytes.Buffer, error) {
	buffer := bytes.NewBuffer(*keys.EdPubKey)
	if err := encoding.WriteVarBytes(buffer, keys.BLSPubKey.Marshal()); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint64(buffer, binary.LittleEndian, amount); err != nil {
		return nil, err
	}

	return buffer, nil
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
