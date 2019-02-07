package core

import (
	"bytes"
	"errors"
	"fmt"
	"sort"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

// AcceptTx attempt to verify a transaction once it is received from
// the network. If the verification passes, this transaction will
// be added to the mempool.
func (b *Blockchain) AcceptTx(tx *transactions.Stealth) error {
	// Check if we already have this in the database first
	key := append(database.TX, tx.R...)
	exists, err := b.db.Has(key)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	// Check if this transaction is coinbase (can not accept it outside of a block)
	if tx.Type == transactions.CoinbaseType {
		return errors.New("can not accept a coinbase transaction outside of a block")
	}

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
	// Version check
	if tx.Version != 0x00 {
		return errors.New("invalid tx: unknown version")
	}

	// Type check
	if tx.Type > 0x05 {
		return errors.New("invalid tx: unrecognized type")
	}

	// Verify type specific info
	if err := b.verifyTypeInfo(tx.TypeInfo); err != nil {
		return err
	}

	// Check if hash is properly calculated
	hash, err := tx.CalculateHash()
	if err != nil {
		return fmt.Errorf("error verifying tx: %v", err)
	}

	if bytes.Compare(hash, tx.R) != 0 {
		return errors.New("invalid tx: hash mismatch")
	}

	return nil
}

// Filter function to identify a payload and verify it properly.
func (b *Blockchain) verifyTypeInfo(pl transactions.TypeInfo) error {
	t := pl.Type()
	switch t {
	case transactions.CoinbaseType:
		c := pl.(*transactions.Coinbase)
		return b.verifyCoinbase(c)
	case transactions.BidType:
		bid := pl.(*transactions.Bid)
		return b.verifyBid(bid)
	case transactions.StakeType:
		s := pl.(*transactions.Stake)
		return b.verifyStake(s)
	case transactions.StandardType:
		s := pl.(*transactions.Standard)
		return b.verifyStandard(s)
	case transactions.TimelockType:
		t := pl.(*transactions.Timelock)
		return b.verifyTimelock(t)
	case transactions.ContractType:
		c := pl.(*transactions.Contract)
		return b.verifyContract(c)
	}
	return nil
}

// AcceptBlock will attempt to verify a block once it is received from
// the network. If the verification passes, the block will be added
// to the database.
func (b *Blockchain) AcceptBlock(blk *block.Block) error {
	// Check if we already have this in the database first
	exists, err := b.db.Has(blk.Header.Hash)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	// Check if previous block hash is correct
	hdr, err := b.db.GetBlockHeaderByHeight(blk.Header.Height - 1)
	if err != nil {
		return err
	}
	prevHeaderHash := hdr.Hash
	if bytes.Compare(blk.Header.PrevBlock, prevHeaderHash) != 0 {
		return errors.New("Invalid block: previous block hash mismatch")
	}

	// Get header from db
	prevBlock, err := b.GetBlock(prevHeaderHash)
	if err != nil {
		return err
	}

	// Height check
	if blk.Header.Height != prevBlock.Header.Height+1 {
		return errors.New("Invalid block: height incorrect")
	}

	// Timestamp check
	if blk.Header.Timestamp < prevBlock.Header.Timestamp {
		return errors.New("Invalid block: timestamp too far in the past")
	}

	// Verify block
	if err := b.VerifyBlock(blk); err != nil {
		return err
	}

	for _, v := range blk.Txs {
		// Clear out all matching entries in mempool
		tx := v.(*transactions.Stealth)
		if b.memPool.Exists(tx.Hex()) {
			b.memPool.RemoveTx(tx)
		}

		// Update provisioners
		if b.provisioner && tx.Type == transactions.StakeType {
			pl := tx.TypeInfo.(*transactions.Stake)
			b.AddProvisionerInfo(tx, pl.Output.Amount)
			b.totalStakeWeight += pl.Output.Amount
		}
	}

	// Add to database
	//if err := b.db.WriteHeaders([]*block.Header{block.Header}); err != nil {
	//	return err
	//}

	if err := b.db.WriteBlockTransactions([]*block.Block{blk}); err != nil {
		return err
	}

	// Update variables
	b.height = blk.Header.Height
	b.round = blk.Header.Height + 1
	b.currSeed = blk.Header.Seed
	b.lastHeader = blk.Header

	if b.provisioner {
		b.UpdateProvisioners()
		b.roundChan <- 1
	}
	// Should update generator merkle tree here as well

	// TODO: Relay
	return nil
}

// VerifyBlock will perform sanity/consensus checks on a block.
func (b *Blockchain) VerifyBlock(blk *block.Block) error {
	// Check hash
	hash := blk.Header.Hash
	if err := blk.SetHash(); err != nil {
		return err
	}

	if bytes.Compare(hash, blk.Header.Hash) != 0 {
		return errors.New("Invalid block: hash mismatch")
	}

	// Check all transactions
	for i, v := range blk.Txs {
		tx := v.(*transactions.Stealth)

		// First transaction has to be coinbase
		if i == 0 {
			if tx.Type != transactions.CoinbaseType {
				return errors.New("first block transaction is not coinbase")
			}
		}

		// Other transactions can not be coinbase
		if i > 0 {
			if tx.Type == transactions.CoinbaseType {
				return errors.New("coinbase transaction found at wrong index")
			}
		}

		if err := b.VerifyTx(tx); err != nil {
			return err
		}
	}

	// Check merkle root
	root := blk.Header.TxRoot
	if err := blk.SetRoot(); err != nil {
		return err
	}

	if bytes.Compare(root, blk.Header.TxRoot) != 0 {
		return errors.New("Invalid block: merkle root mismatch")
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

	latestHeader := &block.Header{}
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
