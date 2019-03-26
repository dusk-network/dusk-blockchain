package chain

import (
	"bytes"
	"encoding/hex"

	"github.com/pkg/errors"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof"
)

var consensusSeconds = 10

// Chain represents the nodes blockchain
// This struct will be aware of the current state of the node.
type Chain struct {
	prevBlock block.Block
	db        Database
}

// AcceptBlock will accept a block if
// 1. We have not seen it before
// 2. All stateless and statefull checks are true
// Returns nil, if checks passed and block was successfully saved
func (c *Chain) AcceptBlock(blk block.Block) error {

	// 1. Check that we have not seen this block before
	err := c.checkBlockExists(blk)
	if err != nil {
		return err
	}

	// 2. Check that stateless and stateful checks pass
	err = c.VerifyBlock(blk)
	if err != nil {
		return err
	}

	// 3. Save block in database
	err = c.writeBlock(blk)
	if err != nil {
		return err
	}

	// 4. Send block to mempool via event-bus. mempool will remove all transactions in block from the mempool

	return nil
}

// VerifyBlock will verify whether a block is valid according to the rules of the consensus
// returns nil if a block is valid
func (c Chain) VerifyBlock(blk block.Block) error {

	if err := c.checkBlockHeader(blk); err != nil {
		return errors.Wrapf(err, " block header verification failed with hash %s", hex.EncodeToString(blk.Header.Hash))
	}

	if err := checkMultiCoinbases(blk.Txs); err != nil {
		return err
	}

	for i, merklePayload := range blk.Txs {
		tx, ok := merklePayload.(transactions.Transaction)
		if !ok {
			return errors.New("tx does not implement the transaction interface")
		}
		if err := c.verifyTX(uint64(i), uint64(blk.Header.Timestamp), tx); err != nil {
			return err
		}
	}
	return nil
}

//VerifyTX will verify whether a transaction is valid by checking:
// - It has not been double spent
// - It is not malformed
// Returns nil if a tx is valid
func (c *Chain) VerifyTX(tx transactions.Transaction) error {

	approxBlockTime := uint64(consensusSeconds) + uint64(c.prevBlock.Header.Timestamp)

	if err := c.verifyTX(0, approxBlockTime, tx); err != nil {
		return err
	}
	return nil
}

//VerifyTX will verify whether a transaction is valid by checking:
// - It has not been double spent
// - It is not malformed
// Index indicates the position that the transaction is in, in a block
// If it is a solo transaction, this is set to 0
// blockTime indicates what time the transaction will be included in a block
// If it is a solo transaction, the blockTime is calculated by using currentBlockTime+consensusSeconds
// Returns nil if a tx is valid
func (c *Chain) verifyTX(index uint64, blockTime uint64, tx transactions.Transaction) error {

	if err := c.checkStandardTx(tx.StandardTX()); err != nil && tx.Type() != transactions.CoinbaseType {
		return err
	}

	if err := c.checkSpecialFields(index, blockTime, tx); err != nil {
		return err
	}

	return nil
}

// checks whether a block header is malformed,
// These are stateless and stateful checks
// returns nil, if all checks pass
func (c Chain) checkBlockHeader(blk block.Block) error {

	// Version
	if blk.Header.Version > 0 {
		return errors.New("unsupported block version")
	}

	// blk.Headerhash = prevHeaderHash
	if !bytes.Equal(blk.Header.PrevBlock, c.prevBlock.Header.Hash) {
		return errors.New("Previous block hash does not equal the previous hash in the current block")
	}

	// blk.Headerheight = prevHeaderHeight +1
	if blk.Header.Height != c.prevBlock.Header.Height+1 {
		return errors.New("current block height is not one plus the previous block height")
	}

	// blk.Timestamp > prevTimestamp
	if blk.Header.Timestamp <= c.prevBlock.Header.Timestamp {
		return errors.New("current timestamp is less than the previous timestamp")
	}

	// Merkle tree check -- Check is here as the root is not calculated on decode
	tR := blk.Header.TxRoot
	if err := blk.SetRoot(); err != nil {
		return errors.New("could not calculate the merkle tree root for this header")
	}

	if !bytes.Equal(tR, blk.Header.TxRoot) {
		return errors.New("merkle root mismatch")
	}

	return nil
}

// Checks whether the standard fields are correct.
// These checks are both stateless and stateful.
func (c Chain) checkStandardTx(tx transactions.Standard) error {
	// Version -- currently we only accept Version 0
	if tx.Version != 0 {
		return errors.New("invalid transaction version")
	}

	// Type - currently we only have five types
	if tx.TxType > 5 {
		return errors.New("invalid transaction type")
	}

	// Inputs - must contain at least one
	if len(tx.Inputs) == 0 {
		return errors.New("transaction must contain atleast one input")
	}

	// Inputs - should not have duplicate key images
	if tx.Inputs.HasDuplicates() {
		return errors.New("there are duplicate key images in this transaction")
	}

	// Outputs - must contain atleast one
	if len(tx.Outputs) == 0 {
		return errors.New("transaction must contain atleast one output")
	}

	// Outputs - should not have duplicate destination keys
	if tx.Outputs.HasDuplicates() {
		return errors.New("there are duplicate destination keys in this transaction")
	}

	// Rangeproof - should be valid
	if err := checkRangeProof(rangeproof.Proof{}); err != nil {
		return err
	}

	// KeyImage - should not be present in the database
	if err := c.checkTXDoubleSpent(tx.Inputs); err != nil {
		return err
	}

	return nil
}

// returns an error if there is more than one coinbase transaction
//  in the list or if there are none
func checkMultiCoinbases(txs []transactions.Transaction) error {

	var seen bool

	for _, tx := range txs {
		if tx.Type() != transactions.CoinbaseType {
			continue
		}
		if seen {
			return errors.New("multiple coinbase transactions present")
		}
		seen = true
	}

	if !seen {
		return errors.New("no coinbase transactions in the list")
	}
	return nil
}

func checkRangeProof(p rangeproof.Proof) error {
	return nil
}

// checks that the transaction has not been spent by checking the database for that key image
// returns nil if item not in database
func (c Chain) checkTXDoubleSpent(inputs transactions.Inputs) error {

	var err error

	// 1. Check keyImage for each input has not been used already
	for _, input := range inputs {
		has, err := c.db.hasKeyImage(input.KeyImage)
		if err != nil {
			return err
		}
		if has {
			return errors.New("keyimage has already been used")
		}
		return err
	}

	return err
}
