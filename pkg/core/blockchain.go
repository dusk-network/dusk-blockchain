package core

import (
	"bytes"
	"errors"
	"fmt"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

// Blockchain defines a processor for blocks and transactions which
// makes sure that any passed data is in line with the current
// consensus, and maintains a memory pool of all known unconfirmed
// transactions. Properly verified transactions and blocks will be
// added to the memory pool and the database respectively.
type Blockchain struct {
	memPool *MemPool
	// database
	// config
	// generating
	// provisioning
}

// NewBlockchain will return a new Blockchain instance with an
// initialized mempool. This Blockchain instance should then be
// ready to process incoming transactions and blocks.
func NewBlockchain() *Blockchain {
	chain := &Blockchain{}
	chain.memPool.Init()

	return chain
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

	// Check for input/output conflicts in mempool and db

	if err := b.VerifyTx(tx); err != nil {
		return err
	}

	b.memPool.AddTx(tx)
	// Relay tx
	return nil
}

// VerifyTx will perform sanity/consensus checks on a transaction.
func (b *Blockchain) VerifyTx(tx *transactions.Stealth) error {
	// Add basic sanity checks for version, type, value

	// Check if coinbase/reward tx, we should not accept those outside
	// of a block.
	// Implement when tx types are properly coded

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
	// Implement when database is added

	// Check if previous block hash is correct
	// Implement when we have database
	// Should probably add a height sanity check as well?

	// Add timestamp comparison between this and last block

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
	// Relay
	return nil
}

// VerifyBlock will perform sanity/consensus checks on a block.
func (b *Blockchain) VerifyBlock(block *payload.Block) error {
	// Check that there is at least one transaction (coinbase/reward) in this block
	if len(block.Txs) == 0 {
		return errors.New("invalid block: no transactions")
	}

	// Check hash
	hash := block.Header.Hash
	if err := block.SetHash(); err != nil {
		return err
	}

	if bytes.Compare(hash, block.Header.Hash) != 0 {
		return errors.New("invalid block: hash mismatch")
	}

	// Check that first transaction is coinbase/reward, and the rest isn't
	// Implement once properly coded

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
