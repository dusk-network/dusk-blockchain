package chain

import (
	"bytes"
	"encoding/binary"
	"math/big"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/zkproof"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

var consensusSeconds = 20

// Chain represents the nodes blockchain
// This struct will be aware of the current state of the node.
type Chain struct {
	eventBus  *wire.EventBus
	PrevBlock block.Block
	db        Database
	bidList   *user.BidList

	// collector channels
	blockChannel <-chan *block.Block
	txChannel    <-chan transactions.Transaction
}

// New returns a new chain object
func New(eventBus *wire.EventBus) (*Chain, error) {
	db, err := NewDatabase("demo", false)
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
	x := zkproof.CalculateX(zkproof.Uint64ToScalar(totalAmount), zkproof.BytesToScalar(tx.M))
	c.bidList.AddBid(x)

	var bidListBytes []byte
	for _, bid := range *c.bidList {
		bidListBytes = append(bidListBytes, bid[:]...)
	}

	c.eventBus.Publish(msg.BidListTopic, bytes.NewBuffer(bidListBytes))
	return nil
}

func getTxTotalOutputAmount(tx transactions.Transaction) (totalAmount uint64) {
	for _, output := range tx.StandardTX().Outputs {
		amount := big.NewInt(0).SetBytes(output.Commitment).Uint64()
		totalAmount += amount
	}

	return
}
