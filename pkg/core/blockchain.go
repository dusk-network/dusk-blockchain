package core

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math"

	log "github.com/sirupsen/logrus"

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

type Config struct {
	UpdateProvisioners func(*block.Block) error
}

const maxLockTime = math.MaxUint16

// Blockchain defines a processor for blocks and transactions which
// makes sure that any passed data is in line with the current
// consensus, and maintains a memory pool of all known unconfirmed
// transactions. Properly verified transactions and blocks will be
// added to the memory pool and the database respectively.
type Blockchain struct {
	// Database
	db *database.BlockchainDB

	cfg *Config

	// Basic fields
	memPool *MemPool
	net     protocol.Magic
	height  uint64

	lastHeader *block.Header // Last validated block on the chain

}

// NewBlockchain returns a new Blockchain instance with an initialized mempool.
// This Blockchain instance should then be ready to process incoming transactions and blocks.
func NewBlockchain(db *database.BlockchainDB, net protocol.Magic, cfg *Config) (*Blockchain, error) {

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

	chain.cfg = cfg

	// Set up mempool and populate struct fields
	chain.memPool = &MemPool{}
	chain.memPool.Init()
	chain.db = db
	chain.net = net

	chain.lastHeader, err = chain.GetLatestHeader()
	if err != nil {
		return nil, err
	}

	chain.height = chain.lastHeader.Height

	return chain, nil
}

// GetLatestHeader gives the latest block header
func (b *Blockchain) GetLatestHeader() (*block.Header, error) {
	return nil, nil
}

// GetHeaders gives block headers from the database, starting and
// stopping at the provided locators.
func (b *Blockchain) GetHeaders(start []byte, stop []byte) ([]*block.Header, error) {
	return nil, nil
}

// GetBlock will return the block from the received hash
func (b *Blockchain) GetBlock(hash []byte) (*block.Block, error) {
	return nil, nil
}

// AddHeaders will add block headers to the chain
func (b *Blockchain) AddHeaders(msg *payload.MsgHeaders) error {
	if err := b.db.WriteHeaders(msg.Headers); err != nil {
		return err
	}
	return nil
}
