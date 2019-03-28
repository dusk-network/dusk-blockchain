package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"net"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/zkproof"

	"github.com/bwesterb/go-ristretto"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/chain"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/factory"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
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

	// holder for blocks during initialization
	Blocks []block.Block
}

type txCollector struct {
	txChannel chan *bytes.Buffer
}

func Setup() *Server {
	eventBus := wire.New()
	keys, _ := user.NewRandKeys()

	committee := committee.NewCommitteeStore(eventBus)
	go committee.Listen()

	chain, err := chain.New(eventBus)
	if err != nil {
		panic(err)
	}
	go chain.Listen()

	stakeChannel := initStakeCollector(eventBus)
	bidChannel := initBidCollector(eventBus)

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
	bid, d, k := makeBid()
	chain.AcceptTx(stake)
	chain.AcceptTx(bid)

	// start consensus factory
	factory := factory.New(eventBus, timeOut, committee, keys, d, k)
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

func (s *Server) Listen() {
	for {
		select {
		case b := <-s.stakeChannel:
			fmt.Println("stake received")
			s.stakes = append(s.stakes, b)
		case b := <-s.bidChannel:
			fmt.Println("bid received")
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
	fmt.Println("attempting to accept connection to ", conn.RemoteAddr().String())
	peer := peer.NewPeer(conn, true, protocol.TestNet, s.eventBus)
	// send the latest block
	buffer := new(bytes.Buffer)
	s.chain.PrevBlock.Encode(buffer)
	peer.Conn.Write(buffer.Bytes())

	if err := peer.Run(); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("we have connected to " + peer.Conn.RemoteAddr().String())

	s.sendStakesAndBids(peer)
}

func (s *Server) OnConnection(conn net.Conn, addr string) {
	fmt.Println("attempting to connect to ", conn.RemoteAddr().String())
	peer := peer.NewPeer(conn, false, protocol.TestNet, s.eventBus)
	// get latest block
	buf := make([]byte, 1024)
	peer.Conn.Read(buf)
	var blk block.Block
	blk.Decode(bytes.NewBuffer(buf))
	s.Blocks = append(s.Blocks, blk)
	if err := peer.Run(); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("we have connected to " + peer.Conn.RemoteAddr().String())

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
	binary.BigEndian.PutUint64(commitment[:32], uint64(outputAmount))
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
	binary.BigEndian.PutUint64(commitment[:32], uint64(outputAmount))
	destKey, _ := crypto.RandEntropy(32)
	rangeProof, _ := crypto.RandEntropy(32)
	output, _ := transactions.NewOutput(commitment, destKey, rangeProof)
	bid.Outputs = transactions.Outputs{output}

	return bid, dScalar, k
}
