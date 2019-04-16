package chain

import (
	"bytes"
	"encoding/binary"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"math/big"

	"github.com/bwesterb/go-ristretto"
	log "github.com/sirupsen/logrus"
	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	_ "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/heavy"
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

	// TODO: exposed for demo, possibly undo later
	PrevBlock block.Block
	BidList   *user.BidList

	// collector channels
	blockChannel <-chan *block.Block
	txChannel    <-chan transactions.Transaction
}

// New returns a new chain object
// TODO: take out demo constructions (db, collectors) and improve it after demo
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
	txChannel := initTxCollector(eventBus)

	return &Chain{
		eventBus:     eventBus,
		db:           db,
		BidList:      &user.BidList{},
		PrevBlock:    *genesisBlock,
		blockChannel: blockChannel,
		txChannel:    txChannel,
	}, nil
}

// Listen to the collectors
func (c *Chain) Listen() {
	for {
		select {
		case blk := <-c.blockChannel:
			c.AcceptBlock(*blk)
		case tx := <-c.txChannel:
			c.AcceptTx(tx)
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

func (c *Chain) propagateTx(tx transactions.Transaction) error {
	buffer := new(bytes.Buffer)

	topicBytes := topics.TopicToByteArray(topics.Tx)
	if _, err := buffer.Write(topicBytes[:]); err != nil {
		return err
	}

	if err := tx.Encode(buffer); err != nil {
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
	c.BidList.AddBid(x)

	c.PropagateBidList()
	return nil
}

func (c *Chain) PropagateBidList() {
	var bidListBytes []byte
	for _, bid := range *c.BidList {
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
