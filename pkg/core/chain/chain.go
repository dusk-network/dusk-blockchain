package chain

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math/big"

	//"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"

	logger "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"

	"github.com/bwesterb/go-ristretto"
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

var log *logger.Entry = logger.WithFields(logger.Fields{"process": "chain"})

// Chain represents the nodes blockchain
// This struct will be aware of the current state of the node.
type Chain struct {
	eventBus *wire.EventBus
	rpcBus   *wire.RPCBus
	db       database.DB

	prevBlock block.Block
	bidList   *user.BidList

	// collector channels
	candidateChan   <-chan *block.Block
	winningHashChan <-chan []byte
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

	// read and decode the genesis block hex
	genesisBlock := block.NewBlock()
	switch cfg.Get().General.Network {
	case "testnet":

		blob, err := hex.DecodeString(cfg.TestNetGenesisBlob)
		if err != nil {
			panic(err)
		}

		var buf bytes.Buffer
		buf.Write(blob)
		if err := genesisBlock.Decode(&buf); err != nil {
			panic(err)
		}
	}

	// set up collectors
	candidateChan := initBlockCollector(eventBus, string(topics.Candidate))
	winningHashChan := initWinningHashCollector(eventBus)

	c := &Chain{
		eventBus:        eventBus,
		rpcBus:          rpcBus,
		db:              db,
		bidList:         &user.BidList{},
		prevBlock:       *genesisBlock,
		candidateChan:   candidateChan,
		winningHashChan: winningHashChan,
	}

	eventBus.SubscribeCallback(string(topics.Block), c.onAcceptBlock)
	return c, nil
}

// Listen to the collectors
func (c *Chain) Listen() {
	for {
		select {

		case b := <-c.candidateChan:
			_ = c.handleCandidateBlock(*b)
		case blockHash := <-c.winningHashChan:
			_ = c.handleWinningHash(blockHash)

		// wire.RPCBus requests handlers
		case r := <-wire.GetLastBlockChan:
			buf := new(bytes.Buffer)
			_ = c.prevBlock.Encode(buf)
			r.RespChan <- *buf
		}
	}
}

func (c *Chain) propagateBlock(blk block.Block) error {
	buffer := new(bytes.Buffer)
	if err := blk.Encode(buffer); err != nil {
		return err
	}

	msg, err := wire.AddTopic(buffer, topics.Block)
	if err != nil {
		return err
	}

	c.eventBus.Stream(string(topics.Gossip), msg)
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

	log.Info("Close database")

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

func (c *Chain) onAcceptBlock(m *bytes.Buffer) error {
	blk := block.NewBlock()
	if err := blk.Decode(m); err != nil {
		return err
	}

	return c.AcceptBlock(*blk)
}

// AcceptBlock will accept a block if
// 1. We have not seen it before
// 2. All stateless and statefull checks are true
// Returns nil, if checks passed and block was successfully saved
func (c *Chain) AcceptBlock(blk block.Block) error {
	field := logger.Fields{"process": "accept block"}
	l := log.WithFields(field)

	l.Trace("procedure started")

	// 1. Check that stateless and stateful checks pass
	if err := verifiers.CheckBlock(c.db, c.prevBlock, blk); err != nil {
		l.Errorf("verification failed: %s", err.Error())
		return err
	}

	// 2. Store block in database
	err := c.db.Update(func(t database.Transaction) error {
		return t.StoreBlock(&blk)
	})

	if err != nil {
		l.Errorf("block storing failed: %s", err.Error())
		return err
	}

	c.prevBlock = blk

	// 3. Notify other subsystems for the accepted block
	// Subsystems listening for this topic:
	// mempool.Mempool
	// consensus.generation.broker
	buf := new(bytes.Buffer)
	if err := blk.Encode(buf); err != nil {
		l.Errorf("block encoding failed: %s", err.Error())
		return err
	}

	c.eventBus.Publish(string(topics.AcceptedBlock), buf)

	// 4. Gossip advertise block Hash
	if err := c.advertiseBlock(blk); err != nil {
		l.Errorf("block advertising failed: %s", err.Error())
		return err
	}

	// 5. Cleanup obsolete candidate blocks
	var count uint32
	err = c.db.Update(func(t database.Transaction) error {
		count, err = t.DeleteCandidateBlocks(blk.Header.Height)
		return err
	})

	if err != nil {
		// Not critical enough to abort the accepting procedure
		log.Warnf("DeleteCandidateBlocks failed with an error: %s", err.Error())
	} else {
		log.Infof("%d deleted candidate blocks", count)
	}

	l.Trace("procedure ended")

	return nil
}

func (c *Chain) AcceptGenesisBlock() error {

	field := logger.Fields{"process": "accept genesis block"}
	l := log.WithFields(field)

	genesisBlock := &block.Block{
		Header: &block.Header{
			Version:       0,
			Height:        0,
			Timestamp:     0,
			PrevBlockHash: []byte{0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 100, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			TxRoot:        []byte{0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 100, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			CertHash:      []byte{0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 100, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			Seed:          []byte{0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 100, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
	}

	err := genesisBlock.SetHash()
	if err != nil {
		l.Errorf("hash calculating failed: %s", err.Error())
		return err
	}

	// 1. Store block in database
	err = c.db.Update(func(t database.Transaction) error {
		return t.StoreBlock(genesisBlock)
	})

	if err != nil {
		l.Errorf("storing failed: %s", err.Error())
		return err
	}

	c.prevBlock = *genesisBlock

	// 2. Notify other subsystems for the accepted block
	// Subsystems listening for this topic:
	// mempool.Mempool
	// consensus.generation.broker
	buf := new(bytes.Buffer)
	if err := genesisBlock.Encode(buf); err != nil {
		l.Errorf("encoding failed: %s", err.Error())
		return err
	}

	c.eventBus.Publish(string(topics.AcceptedBlock), buf)

	return nil
}

func (c *Chain) handleCandidateBlock(candidate block.Block) error {
	// Ensure the candidate block satisfies all chain rules
	if err := verifiers.CheckBlock(c.db, c.prevBlock, candidate); err != nil {
		log.Errorf("verifying the candidate block failed: %s", err.Error())
		return err
	}

	// Save it into persistent storage
	err := c.db.Update(func(t database.Transaction) error {
		err := t.StoreCandidateBlock(&candidate)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Errorf("storing the candidate block failed: %s", err.Error())
		return err
	}

	return nil
}

func (c *Chain) handleWinningHash(blockHash []byte) error {
	// Fetch the candidate block that the winningHash points at
	var candidate *block.Block
	err := c.db.View(func(t database.Transaction) error {
		var err error
		candidate, err = t.FetchCandidateBlock(blockHash)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Errorf("fetching a candidate block failed: %s", err.Error())
		return err
	}

	// Run the general procedure of block accepting
	return c.AcceptBlock(*candidate)
}

// Send Inventory message to all peers
func (c *Chain) advertiseBlock(b block.Block) error {

	msg := &peermsg.Inv{}
	msg.InvList = make([]peermsg.InvVect, 1)
	msg.InvList[0].Type = peermsg.InvTypeBlock
	msg.InvList[0].Hash = b.Header.Hash

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		panic(err)
	}

	withTopic, err := wire.AddTopic(buf, topics.Inv)
	if err != nil {
		return err
	}

	c.eventBus.Stream(string(topics.Gossip), withTopic)
	return nil
}
