package core

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
<<<<<<< HEAD
	"math"
=======
	log "github.com/sirupsen/logrus"
>>>>>>> 7c061916bb7ed43cbbc1b0dd6d5d1a1bda7bcda2
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
	cnf "github.com/spf13/viper"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

var (
	errNoBlockchainDb    = errors.New("Blockchain database is not available")
	errInitialisedCheck  = errors.New("Failed to check if blockchain db is already initialised")
	errBlockValidation   = errors.New("Block failed sanity check")
	errBlockVerification = errors.New("Block failed to be consistent with the current blockchain")

	candidateTimer = 60 * time.Second
	maxLockTime    = math.MaxUint16
)

// Blockchain defines a processor for blocks and transactions which
// makes sure that any passed data is in line with the current
// consensus, and maintains a memory pool of all known unconfirmed
// transactions. Properly verified transactions and blocks will be
// added to the memory pool and the database respectively.
type Blockchain struct {
	// Basic fields
	memPool *MemPool
	net     protocol.Magic
	height  uint64

	// Consensus related
	currSeed   []byte               // Seed of the current round of consensus
	round      uint64               // Current round (block height + 1)
	lastHeader *payload.BlockHeader // Last validated block on the chain
	quitChan   chan int             // Channel used to stop consensus loops
	roundChan  chan int             // Channel used to signify start of a new round
	ctx        *consensus.Context   // Consensus context object

	// Block generator related fields
	generator bool
	bidWeight uint64

	// Provisioner related fields
	provisioner bool
	provPubKey  []byte // Our public key used for provisioning

	reductionChan  chan *payload.MsgReduction
	binaryChan     chan *payload.MsgBinary
	candidateChan  chan *payload.MsgScore
	sigSetChan     chan *payload.MsgSignatureSet
	sigSetVoteChan chan *payload.MsgSigSetVote

	stakeWeight      uint64 // The amount of DUSK staked by the node
	totalStakeWeight uint64 // The total amount of DUSK staked
}

// NewBlockchain returns a new Blockchain instance with an initialized mempool.
// This Blockchain instance should then be ready to process incoming transactions and blocks.
func NewBlockchain(net protocol.Magic) (*Blockchain, error) {
	db := database.GetInstance()

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
			b := payload.NewBlock()
			if err := b.Decode(r); err != nil {
				log.WithField("prefix", "blockchain").Error("Failed to add genesis block header to db")
				db.Delete(marker)
				return nil, err
			}

			err = db.WriteHeaders([]*payload.BlockHeader{b.Header})
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

	chain := &Blockchain{}

	// Set up mempool and populate struct fields
<<<<<<< HEAD
	chain.memPool = &MemPool{}
	chain.memPool.Init()
	chain.db = db
	chain.net = net

	// Consensus set-up
	chain.reductionChan = make(chan *payload.MsgReduction, 200)
	chain.binaryChan = make(chan *payload.MsgBinary, 200)
	chain.sigSetChan = make(chan *payload.MsgSignatureSet, 200)
	chain.sigSetVoteChan = make(chan *payload.MsgSigSetVote, 200)
	chain.candidateChan = make(chan *payload.MsgScore, 1)
	chain.quitChan = make(chan int, 1)
	chain.roundChan = make(chan int, 1)
	chain.lastHeader, err = chain.GetLatestHeader()
=======
	//chain.memPool.Init() //TODO: TV commented this line because of no memPool instance (yet)
	chain.net = net

	// Consensus set-up
	chain.reductionChan = make(chan *payload.MsgReduction)
	chain.binaryChan = make(chan *payload.MsgBinary)
	chain.candidateChan = make(chan *payload.MsgScore)
	chain.quitChan = make(chan int)
	chain.roundChan = make(chan int)
	chain.lastHeader, err = db.GetLatestHeader()
>>>>>>> 7c061916bb7ed43cbbc1b0dd6d5d1a1bda7bcda2
	if err != nil {
		return nil, err
	}

	chain.height = chain.lastHeader.Height
	chain.round = chain.height + 1
	chain.currSeed = chain.lastHeader.Seed

	return chain, nil
}

// Loop function for provisioner nodes
func (b *Blockchain) provisionerLoop() {
	b.provisioner = true

	for {
		select {
		case <-b.quitChan:
			b.provisioner = false
			return
		case <-b.roundChan:
			// Should add a check here if we're staking, and if not
			// then create a staking transaction

			b.ctx.Reset()
			timer := time.NewTimer(candidateTimer)
			var finalHash []byte
			select {
			case <-timer.C:
				b.ctx.Empty = true
			case m := <-b.candidateChan:
				timer.Stop()
				b.ctx.BlockHash = m.CandidateHash
			}

			if err := consensus.BlockReduction(b.ctx, b.reductionChan); err != nil {
				// Log
				b.provisioner = false
				return
			}

			if err := consensus.BinaryAgreement(b.ctx, b.binaryChan); err != nil {
				// Log
				b.provisioner = false
				return
			}

			if !b.ctx.Empty {
				if err := consensus.SignatureSetReduction(b.ctx, b.sigSetChan, b.sigSetVoteChan); err != nil {
					// Log
					b.provisioner = false
					return
				}

				if bytes.Equal(finalHash, b.ctx.BlockHash) {
					// send final block with set of signatures
					break
				}

				// send tentative block with set of signatures
				break
			}

			// send tentative block without set of signatures
			break
		}
	}
}

// Placeholder at the moment, just to get structure down
//func (b *Blockchain) generatorLoop() {
//	b.generator = true
//
//	for {
//		select {
//		case <-b.quitChan:
//			b.generator = false
//			return
//		case <-b.roundChan:
//			if err := b.Generate(nil); err != nil {
//				// Log
//				b.generator = false
//				return
//			}
//		}
//	}
//}

// AcceptTx attempt to verify a transaction once it is received from
// the network. If the verification passes, this transaction will
// be added to the mempool.
func (b *Blockchain) AcceptTx(tx *transactions.Stealth) error {
	// Check if we already have this in the database first
	// Implement when database is added

	// Check if this transaction is already in the mempool
	if b.memPool.Exists(tx.Hex()) {
		return errors.New("duplicate tx")
	}

	if err := b.VerifyTx(tx); err != nil {
		return err
	}

	b.memPool.AddTx(tx)
	// Relay tx
	return nil
}

// VerifyTx will perform sanity/consensus checks on a transaction.
func (b *Blockchain) VerifyTx(tx *transactions.Stealth) error {
	if tx.Version != 0x00 {
		return fmt.Errorf("invalid tx: unknown version")
	}

	if tx.Type != transactions.StandardType && tx.Type != transactions.BidType {
		return fmt.Errorf("invalid tx: unknown type")
	}

	// Verify inputs and outputs (values, signatures)
	// Implement once these are properly coded

	// Check if hash is properly calculated
	hash, err := tx.CalculateHash()
	if err != nil {
		return fmt.Errorf("error verifying tx: %v", err)
	}

	if bytes.Compare(hash, tx.Hash) != 0 {
		return errors.New("invalid tx: hash mismatch")
	}

	return nil
}

// AcceptBlock will attempt to verify a block once it is received from
// the network. If the verification passes, the block will be added
// to the database.
func (b *Blockchain) AcceptBlock(block *payload.Block) error {
	// Check if we already have this in the database first
	db := database.GetInstance()
	exists, err := db.Has(block.Header.Hash)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	// Check if previous block hash is correct
	hdr, err := db.GetBlockHeaderByHeight(block.Header.Height - 1)
	if err != nil {
		return err
	}
	prevHeaderHash := hdr.Hash
	if bytes.Compare(block.Header.PrevBlock, prevHeaderHash) != 0 {
		return errors.New("Invalid block: previous block hash mismatch")
	}

	// Get header from db
	prevBlock, err := b.GetBlock(prevHeaderHash)
	if err != nil {
		return err
	}

	// Height check
	if block.Header.Height != prevBlock.Header.Height+1 {
		return errors.New("Invalid block: height incorrect")
	}

	// Timestamp check
	if block.Header.Timestamp < prevBlock.Header.Timestamp {
		return errors.New("Invalid block: timestamp too far in the past")
	}

	// Verify block
	if err := b.VerifyBlock(block); err != nil {
		return err
	}

	// Clear out all matching entries in mempool
<<<<<<< HEAD
	for _, v := range block.Txs {
		tx := v.(*transactions.Stealth)
		if b.memPool.Exists(tx.Hex()) {
			b.memPool.RemoveTx(tx)
		}

		if b.provisioner {
			// Update provisioners
			if tx.Type == transactions.StakeType {
				var amount uint64
				for _, output := range tx.Outputs {
					amount += output.Amount
				}

				b.totalStakeWeight += amount
			}
		}
	}
=======
	//for _, v := range block.Txs {
	//	tx := v.(*transactions.Stealth)
	//	if b.memPool.Exists(tx.Hex()) {
	//		b.memPool.RemoveTx(tx)
	//	}
	//}
>>>>>>> 7c061916bb7ed43cbbc1b0dd6d5d1a1bda7bcda2

	// Add to database
	//if err := db.WriteHeaders([]*payload.BlockHeader{block.Header}); err != nil {
	//	return err
	//}

	if err := db.WriteBlockTransactions([]*payload.Block{block}); err != nil {
		return err
	}

	// Update variables
	b.height = block.Header.Height
	b.round = block.Header.Height + 1
	b.currSeed = block.Header.Seed
	b.lastHeader = block.Header

	if b.provisioner {
		b.roundChan <- 1
	}
	// Should update generator merkle tree here as well

	// TODO: Relay
	return nil
}

// VerifyBlock will perform sanity/consensus checks on a block.
func (b *Blockchain) VerifyBlock(block *payload.Block) error {
	// Check hash
	hash := block.Header.Hash
	if err := block.SetHash(); err != nil {
		return err
	}

	if bytes.Compare(hash, block.Header.Hash) != 0 {
		return errors.New("Invalid block: hash mismatch")
	}

	// Check all transactions
	for _, v := range block.Txs {
		tx := v.(*transactions.Stealth)
		if err := b.VerifyTx(tx); err != nil {
			return err
		}
	}

	// Check merkle root
	root := block.Header.TxRoot
	if err := block.SetRoot(); err != nil {
		return err
	}

	if bytes.Compare(root, block.Header.TxRoot) != 0 {
		return errors.New("Invalid block: merkle root mismatch")
	}

	return nil
}

// GetLatestHeader gives the latest block header
func (b *Blockchain) GetLatestHeader() (*payload.BlockHeader, error) {
	db := database.GetInstance()

	return db.GetLatestHeader()
}

// GetHeaders gives block headers from the database, starting and
// stopping at the provided locators.
func (b *Blockchain) GetHeaders(start []byte, stop []byte) ([]*payload.BlockHeader, error) {
	db := database.GetInstance()

	return db.ReadHeaders(start, stop)
}

// GetBlock will return the block from the received hash
func (b *Blockchain) GetBlock(hash []byte) (*payload.Block, error) {
	bd := database.GetInstance()

	return bd.GetBlock(hash)
}

// AddHeaders will add block headers to the chain.
func (b *Blockchain) AddHeaders(msg *payload.MsgHeaders) error {
	db := database.GetInstance()
	if err := db.WriteHeaders(msg.Headers); err != nil {
		return err
	}
	return nil
}

// ValidateHeaders will validate headers that were received through the wire.
// TODO: Centralize validation rules
func (b *Blockchain) ValidateHeaders(msg *payload.MsgHeaders) error {
	db := database.GetInstance()
	table := database.NewTable(db, database.HEADER)
	latestHash, err := db.Get(database.LATESTHEADER)
	if err != nil {
		return err
	}

	key := latestHash
	headerBytes, err := table.Get(key)

	latestHeader := &payload.BlockHeader{}
	err = latestHeader.Decode(bytes.NewReader(headerBytes))
	if err != nil {
		return err
	}

	// Sort the headers
	sortedHeaders := msg.Headers
	sort.Slice(sortedHeaders,
		func(i, j int) bool {
			return sortedHeaders[i].Height < sortedHeaders[j].Height
		})

	// Do checks on headers
	for _, currentHeader := range sortedHeaders {

		if latestHeader == nil {
			// This should not happen as genesis header is added if new
			// database, however we check nonetheless
			return errors.New("Previous header is nil")
		}

		// Check current hash links with previous
		if !bytes.Equal(currentHeader.PrevBlock, latestHeader.Hash) {
			return errors.New("Last header hash != current header previous hash")
		}

		// Check current Index is one more than the previous Index
		if currentHeader.Height != latestHeader.Height+1 {
			return errors.New("Last header height != current header height")
		}

		// Check current timestamp is later than the previous header's timestamp
		// TODO: This check implies that we use one central time on all nodes in whatever timezone.
		//  But even then, because of communication delays this will lead to problems.
		//  Let's not do this check!!
		//if latestHeader.Timestamp > currentHeader.Timestamp {
		//	return errors.New("Timestamp of Previous Header is later than Timestamp of current Header")
		//}

		// NOTE: These are the only non-contextual checks we can do without the blockchain state
		latestHeader = currentHeader
	}
	return nil
}

// StartProvisioning will set the node to provisioner status,
// and will start participating in block reduction and binary agreement
// phases of the protocol.
func (b *Blockchain) StartProvisioning() error {
	// Can't generate and provision at the same time
	if b.generator {
		return errors.New("can't start provisioning: currently generating blocks")
	}

	if b.provisioner {
		return errors.New("already provisioning")
	}

	keys, err := consensus.NewRandKeys()
	if err != nil {
		return err
	}

	ctx, err := consensus.NewProvisionerContext(b.totalStakeWeight, b.round, b.currSeed, b.net, keys)
	if err != nil {
		return err
	}

	b.ctx = ctx
	go b.provisionerLoop()
	return nil
}

// StopProvisioning will stop the provisioning process
func (b *Blockchain) StopProvisioning() error {
	if !b.provisioner {
		return errors.New("not provisioning")
	}

	b.quitChan <- 1
	return nil
}
