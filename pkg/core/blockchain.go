package core

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math"

	log "github.com/sirupsen/logrus"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

var (
	errNoBlockchainDb    = errors.New("Blockchain database is not available")
	errInitialisedCheck  = errors.New("Failed to check if blockchain db is already initialised")
	errBlockValidation   = errors.New("Block failed sanity check")
	errBlockVerification = errors.New("Block failed to be consistent with the current blockchain")
)

const maxLockTime = math.MaxUint16

// Blockchain defines a processor for blocks and transactions which
// makes sure that any passed data is in line with the current
// consensus, and maintains a memory pool of all known unconfirmed
// transactions. Properly verified transactions and blocks will be
// added to the memory pool and the database respectively.
type Blockchain struct {
	// Database
	db *database.BlockchainDB

	// Basic fields
	memPool *MemPool
	net     protocol.Magic
	height  uint64
	db      *database.BlockchainDB

	// Consensus related
	currSeed      []byte                     // Seed of the current round of consensus
	round         uint64                     // Current round (block height + 1)
	tau           uint64                     // Current generator threshold
	lastHeader    *block.Header              // Last validated block on the chain
	roundChan     chan int                   // Channel used to signify start of a new round
	ctx           *consensus.Context         // Consensus context object
	consensusChan chan *payload.MsgConsensus // Channel for consensus messages

	// Block generator related fields
	generator bool
	bidWeight uint64

	// Provisioner related fields
	provisioner      bool
	stakeWeight      uint64 // The amount of DUSK staked by the node
	totalStakeWeight uint64 // The total amount of DUSK staked
	provisioners     Provisioners
}

// NewBlockchain returns a new Blockchain instance with an initialized mempool.
// This Blockchain instance should then be ready to process incoming transactions and blocks.
func NewBlockchain(db *database.BlockchainDB, net protocol.Magic) (*Blockchain, error) {
	//path := config.EnvNetCfg.DatabaseDirPath
	//db, err := database.NewBlockchainDB(path)
	//log.WithField("prefix", "database").Debugf("Path to database: %s", path)
	//if err != nil {
	//	log.WithField("prefix", "database").Fatalf("Failed to find db path: %s", path)
	//	return nil, err
	//}

	marker := []byte("HasBeenInitialisedAlready")
	init, err := db.Has(marker)
	if err != nil {
		return nil, errInitialisedCheck
	}

	if !init {
		// This is a new db, so initialise it
		log.WithField("prefix", "blockchain").Info("New Blockchain database initialisation")
		db.Put(marker, []byte{})

		// Add Genesis block (No transactions in Genesis block)
		if net == protocol.DevNet {
			genesisBlock, err := hex.DecodeString(GenesisBlock)
			if err != nil {
				log.WithField("prefix", "blockchain").Error("Failed to add genesis block header to db")
				db.Delete(marker)
				return nil, err
			}

			r := bytes.NewReader(genesisBlock)
			b := block.NewBlock()
			if err := b.Decode(r); err != nil {
				log.WithField("prefix", "blockchain").Error("Failed to add genesis block header to db")
				db.Delete(marker)
				return nil, err
			}

			err = db.WriteHeaders([]*block.Header{b.Header})
			if err != nil {
				log.WithField("prefix", "blockchain").Error("Failed to add genesis block header")
				db.Delete(marker)
				return nil, err
			}
		}

		if net == protocol.TestNet {
			fmt.Println("TODO: Setup the genesisBlock for TestNet")
			return nil, nil
		}

		if net == protocol.MainNet {
			fmt.Println("TODO: Setup the genesisBlock for MainNet")
			return nil, nil
		}
	}

	chain := &Blockchain{db: db}

	// Set up mempool and populate struct fields
	chain.memPool = &MemPool{}
	chain.memPool.Init()
	chain.db = db
	chain.net = net

	// Consensus set-up
	chain.consensusChan = make(chan *payload.MsgConsensus, 500)
	chain.roundChan = make(chan int, 1)
	chain.lastHeader, err = chain.GetLatestHeader()
	if err != nil {
		return nil, err
	}

	chain.height = chain.lastHeader.Height
	chain.round = chain.height + 1
	chain.currSeed = chain.lastHeader.Seed

	// Start consensus loop
	go chain.segregatedByzantizeAgreement()
	return chain, nil
}

// GetLatestHeader gives the latest block header
func (b *Blockchain) GetLatestHeader() (*block.Header, error) {
	db := database.GetInstance()

	return db.GetLatestHeader()
}

// GetHeaders gives block headers from the database, starting and
// stopping at the provided locators.
func (b *Blockchain) GetHeaders(start []byte, stop []byte) ([]*block.Header, error) {
	db := database.GetInstance()

	return db.ReadHeaders(start, stop)
}

// GetBlock will return the block from the received hash
func (b *Blockchain) GetBlock(hash []byte) (*block.Block, error) {
	bd := database.GetInstance()

	return bd.GetBlock(hash)
}

// AddHeaders will add block headers to the chain
func (b *Blockchain) AddHeaders(msg *payload.MsgHeaders) error {
	if err := b.db.WriteHeaders(msg.Headers); err != nil {
		return err
	}
	return nil
}
