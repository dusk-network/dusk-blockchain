package chain

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"sync"

	//"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"

	"github.com/bwesterb/go-ristretto"
	log "github.com/sirupsen/logrus"
	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	_ "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/heavy"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/mempool"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"gitlab.dusk.network/dusk-core/zkproof"
)

var consensusSeconds = 20

// Chain represents the nodes blockchain
// This struct will be aware of the current state of the node.
type Chain struct {
	eventBus *wire.EventBus
	db       database.DB

	// TODO: exposed for demo, undo later
	// We expose prevBlock here so that we can change and retrieve the
	// current height of the chain. This makes jump-in a lot easier while
	// we don't yet store blocks.
	sync.RWMutex
	PrevBlock block.Block
	bidList   *user.BidList

	blockChannel <-chan *block.Block

	// collector channels
	roundChan <-chan uint64
	blockChan <-chan *block.Block
	txChan    <-chan transactions.Transaction

	m *mempool.Mempool
}

// New returns a new chain object
func New(eventBus *wire.EventBus) (*Chain, error) {

	drvr, err := database.From(cfg.Get().Database.Driver)

	if err != nil {
		return nil, err
	}

	db, err := drvr.Open(cfg.Get().Database.Dir, protocol.MagicFromConfig(), false)

	if err != nil {
		return nil, err
	}

	genesisBlock := &block.Block{
		Header: &block.Header{
			Version:   0,
			Height:    0,
			Timestamp: 0,
			PrevBlock: []byte{0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 100, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			TxRoot:    []byte{0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 100, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			CertHash:  []byte{0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 100, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			Seed:      []byte{0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 100, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
	}
	err = genesisBlock.SetHash()
	if err != nil {
		return nil, err
	}

	// set up collectors
	blockChannel := initBlockCollector(eventBus)

	c := &Chain{
		eventBus:     eventBus,
		db:           db,
		bidList:      &user.BidList{},
		PrevBlock:    *genesisBlock,
		blockChannel: blockChannel,
	}

	var verifyTx = func(tx transactions.Transaction) error {
		approxBlockTime := uint64(consensusSeconds) + uint64(c.PrevBlock.Header.Timestamp)
		return c.verifyTX(0, approxBlockTime, tx)
	}

	c.m = mempool.NewMempool(eventBus, verifyTx)

	return c, nil
}

// Listen to the collectors
func (c *Chain) Listen() {
	for {
		select {
		case round := <-c.roundChan:
			// TODO: this is added for testnet jump-in purposes while we dont yet
			// store blocks. remove this later

			// The prevBlock height should be set to round - 1, as the round denotes the
			// height of the block that is currently being forged.
			c.Lock()
			c.PrevBlock.Header.Height = round - 1
			c.Unlock()
		case blk := <-c.blockChan:
			c.AcceptBlock(*blk)
		case r := <-wire.GetLastBlockChan:
			buf := new(bytes.Buffer)
			_ = c.PrevBlock.Encode(buf)
			r.Resp <- *buf
		}
	}
}

func (c *Chain) propagateBlock(blk block.Block) error {
	buffer := new(bytes.Buffer)

	topicBytes := topics.TopicToByteArray(topics.Block)
	if _, err := buffer.Write(topicBytes[:]); err != nil {
		return err
	}

	if err := blk.Encode(buffer); err != nil {
		return err
	}

	c.eventBus.Publish(string(topics.Gossip), buffer)
	return nil
}

func (c *Chain) addProvisioner(tx *transactions.Stake) error {
	buffer := bytes.NewBuffer(tx.PubKeyEd)
	if err := encoding.WriteVarBytes(buffer, tx.PubKeyBLS); err != nil {
		return err
	}

	totalAmount := getTxTotalOutputAmount(tx)
	if err := encoding.WriteUint64(buffer, binary.LittleEndian, totalAmount); err != nil {
		return err
	}

	c.eventBus.Publish(msg.NewProvisionerTopic, buffer)
	return nil
}

func (c *Chain) addBidder(tx *transactions.Bid) error {
	totalAmount := getTxTotalOutputAmount(tx)
	x := calculateX(totalAmount, tx.M)
	c.bidList.AddBid(x)

	c.propagateBidList()
	return nil
}

func (c *Chain) propagateBidList() {
	var bidListBytes []byte
	for _, bid := range *c.bidList {
		bidListBytes = append(bidListBytes, bid[:]...)
	}

	c.eventBus.Publish(msg.BidListTopic, bytes.NewBuffer(bidListBytes))
}

func (c *Chain) Close() error {

	log.WithFields(log.Fields{
		"process": "chain",
	}).Info("Close database")

	drvr, err := database.From(cfg.Get().Database.Driver)

	if err != nil {
		return err
	}

	return drvr.Close()
}

func getTxTotalOutputAmount(tx transactions.Transaction) (totalAmount uint64) {
	for _, output := range tx.StandardTX().Outputs {
		amount := big.NewInt(0).SetBytes(output.Commitment).Uint64()
		totalAmount += amount
	}

	return
}

func calculateX(d uint64, m []byte) user.Bid {
	dScalar := ristretto.Scalar{}
	dScalar.SetBigInt(big.NewInt(0).SetUint64(d))

	mScalar := ristretto.Scalar{}
	mScalar.UnmarshalBinary(m)

	x := zkproof.CalculateX(dScalar, mScalar)

	var bid user.Bid
	copy(bid[:], x.Bytes()[:])
	return bid
}
