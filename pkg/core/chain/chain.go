package chain

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"sync"
	"time"

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
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/verifiers"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"gitlab.dusk.network/dusk-core/zkproof"
)

// Chain represents the nodes blockchain
// This struct will be aware of the current state of the node.
type Chain struct {
	eventBus *wire.EventBus
	rpcBus   *wire.RPCBus
	db       database.DB

	sync.RWMutex
	prevBlock block.Block
	bidList   *user.BidList

	blockChannel <-chan *block.Block

	// collector channels
	blockChan <-chan *block.Block
}

// New returns a new chain object
func New(eventBus *wire.EventBus, rpcBus *wire.RPCBus) (*Chain, error) {
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
		rpcBus:       rpcBus,
		db:           db,
		bidList:      &user.BidList{},
		prevBlock:    *genesisBlock,
		blockChannel: blockChannel,
	}

	return c, nil
}

// Listen to the collectors
func (c *Chain) Listen() {
	for {
		select {
		case blk := <-c.blockChan:
			c.AcceptBlock(*blk)
		case r := <-wire.GetLastBlockChan:
			buf := new(bytes.Buffer)
			_ = c.prevBlock.Encode(buf)
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

// AcceptBlock will accept a block if
// 1. We have not seen it before
// 2. All stateless and statefull checks are true
// Returns nil, if checks passed and block was successfully saved
func (c *Chain) AcceptBlock(blk block.Block) error {

	// 1. Check that stateless and stateful checks pass
	if err := verifiers.CheckBlock(c.db, c.prevBlock, blk); err != nil {
		return err
	}

	// 2. Save block in database
	err := c.db.Update(func(t database.Transaction) error {
		return t.StoreBlock(&blk)
	})

	if err != nil {
		return err
	}

	c.prevBlock = blk

	// 3. Notify other subsystems for the accepted block
	buf := new(bytes.Buffer)
	if err := blk.Encode(buf); err != nil {
		return err
	}

	c.eventBus.Publish(string(topics.AcceptedBlock), buf)

	// 4. Gossip block
	if err := c.propagateBlock(blk); err != nil {
		return err
	}

	return nil
}

// CreateProposalBlock creates a candidate block to be proposed on next round.
func (c *Chain) CreateProposalBlock() (*block.Block, error) {

	// TODO Missing fields for forging the block
	// - Seed
	// - CertHash

	// Retrieve latest verified transactions from Mempool
	r, err := c.rpcBus.Call(wire.GetVerifiedTxs, wire.NewRequest(bytes.Buffer{}, 10))
	if err != nil {
		return nil, err
	}

	lTxs, err := encoding.ReadVarInt(&r)
	if err != nil {
		return nil, err
	}

	txs, err := transactions.FromReader(&r, lTxs)
	if err != nil {
		return nil, err
	}

	// TODO: guard the prevBlock with mutex
	nextHeight := c.prevBlock.Header.Height + 1
	prevHash := c.prevBlock.Header.Hash

	h := &block.Header{
		Version:   0,
		Timestamp: time.Now().Unix(),
		Height:    nextHeight,
		PrevBlock: prevHash,
		TxRoot:    nil,

		Seed:     nil,
		CertHash: nil,
	}

	// Generate the candidate block
	b := &block.Block{
		Header: h,
		Txs:    txs,
	}

	// Update TxRoot
	if err := b.SetRoot(); err != nil {
		return nil, err
	}

	// Generate the block hash
	if err := b.SetHash(); err != nil {
		return nil, err
	}

	// Ensure the forged block satisfies all chain rules
	if err := verifiers.CheckBlock(c.db, c.prevBlock, *b); err != nil {
		return nil, err
	}

	// Save it into persistent storage
	err = c.db.Update(func(t database.Transaction) error {
		err := t.StoreCandidateBlock(b)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return b, nil
}
