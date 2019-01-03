package core

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

var (
	ErrNoBlockchainDb    = errors.New("blockchain database is not available")
	ErrInitialisedCheck  = errors.New("could not check if blockchain db is already initialised")
	ErrBlockValidation   = errors.New("block failed sanity check")
	ErrBlockVerification = errors.New("block failed to be consistent with the current blockchain")
)

// Blockchain defines a processor for blocks and transactions which
// makes sure that any passed data is in line with the current
// consensus, and maintains a memory pool of all known unconfirmed
// transactions. Properly verified transactions and blocks will be
// added to the memory pool and the database respectively.
type Blockchain struct {
	// Basic stuff
	memPool *MemPool
	db      *database.BlockchainDB
	net     protocol.Magic
	height  uint64

	// User info, for sending transactions and participating
	// in the network validation.
	BLSPubKey    *bls.PublicKey
	BLSSecretKey *bls.SecretKey
	// EdPubKey
	// EdSecretKey

	// Block generator stuff
	generator bool
	bidWeight uint64

	// Provisioner stuff
	provisioner      bool
	reductionChan    chan *payload.MsgReduction
	stakeWeight      uint64
	totalStakeWeight uint64
}

// NewBlockchain will return a new Blockchain instance with an
// initialized mempool. This Blockchain instance should then be
// ready to process incoming transactions and blocks.
func NewBlockchain(net protocol.Magic) (*Blockchain, error) {
	db, err := database.NewBlockchainDB("test") // TODO: external path configuration
	if err != nil {
		return nil, ErrNoBlockchainDb
	}

	marker := []byte("HasBeenInitialisedAlready")
	init, err := db.Has(marker)
	if err != nil {
		return nil, ErrInitialisedCheck
	}

	if !init {
		// This is a new db, so initialise it
		db.Put(marker, []byte{})

		// Add genesis block
		if net == protocol.MainNet {
			genesisBlock, err := hex.DecodeString(GenesisBlock)
			if err != nil {
				db.Delete(marker)
				return nil, err
			}

			r := bytes.NewReader(genesisBlock)
			b := payload.NewBlock()
			if err := b.Decode(r); err != nil {
				db.Delete(marker)
				return nil, err
			}

			if err := db.AddHeader(b.Header); err != nil {
				db.Delete(marker)
				return nil, err
			}

			if err := db.AddBlockTransactions(b); err != nil {
				db.Delete(marker)
				return nil, err
			}
		}

		if net == protocol.TestNet {
			fmt.Println("TODO: Setup the genesisBlock for TestNet")
			return nil, nil
		}
	}

	chain := &Blockchain{}

	// Set up mempool and populate struct fields
	chain.memPool.Init()
	chain.db = db
	chain.net = net
	chain.reductionChan = make(chan *payload.MsgReduction)

	return chain, nil
}

func (b *Blockchain) provisionerLoop() {

}

func (b *Blockchain) generatorLoop() {

}

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
	exists, err := b.db.Has(block.Header.Hash)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	// Check if previous block hash is correct
	prevHeaderHash, err := b.GetLatestHeaderHash()
	if err != nil {
		return err
	}

	if bytes.Compare(block.Header.PrevBlock, prevHeaderHash) != 0 {
		return errors.New("invalid block: previous block hash mismatch")
	}

	// Get header from db
	prevHeader, err := b.GetLatestHeader()
	if err != nil {
		return err
	}

	// Height check
	if block.Header.Height != prevHeader.Height+1 {
		return errors.New("invalid block: heightn incorrect")
	}

	// Timestamp check
	if block.Header.Timestamp <= prevHeader.Timestamp {
		return errors.New("invalid block: timestamp too far in the past")
	}

	// Verify block
	if err := b.VerifyBlock(block); err != nil {
		return err
	}

	// Clear out all matching entries in mempool
	for _, v := range block.Txs {
		tx := v.(*transactions.Stealth)
		if b.memPool.Exists(tx.Hex()) {
			b.memPool.RemoveTx(tx)
		}
	}

	// Add to database
	if err := b.db.AddHeader(block.Header); err != nil {
		return err
	}

	if err := b.db.AddBlockTransactions(block); err != nil {
		return err
	}

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
		return errors.New("invalid block: hash mismatch")
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
		return errors.New("invalid block: merkle root mismatch")
	}

	return nil
}

//addHeaders is not safe for concurrent access
func (b *Blockchain) ValidateHeaders(msg *payload.MsgHeaders) error {
	table := database.NewTable(b.db, database.HEADER)
	latestHash, err := b.db.Get(database.LATESTHEADER)
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

func (b *Blockchain) AddHeaders(msg *payload.MsgHeaders) error {
	for _, header := range msg.Headers {
		if err := b.db.AddHeader(header); err != nil {
			return err
		}
	}
	return nil
}

func (b *Blockchain) GetHeaders(start []byte, stop []byte) ([]*payload.BlockHeader, error) {
	return b.db.ReadHeaders(start, stop)
}

func (b *Blockchain) GetLatestHeaderHash() ([]byte, error) {
	return b.db.Get(database.LATESTHEADER)
}

func (b *Blockchain) GetLatestHeader() (*payload.BlockHeader, error) {
	prevHeaderHash, err := b.GetLatestHeaderHash()
	if err != nil {
		return nil, err
	}

	prevHeaderBytes, err := b.db.Get(prevHeaderHash)
	if err != nil {
		return nil, err
	}

	prevHeaderBuf := bytes.NewReader(prevHeaderBytes)
	prevHeader := &payload.BlockHeader{}
	if err := prevHeader.Decode(prevHeaderBuf); err != nil {
		return nil, err
	}

	return prevHeader, nil
}
