package chain

import (
	"bytes"
	"encoding/hex"

	"github.com/pkg/errors"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

// Chain represents the nodes blockchain
// This struct will be aware of the current state of the node.
type Chain struct {
	prevBlock block.Block
	db        database.DB
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

	// 1. Check whether a block is malformed - stateless
	err := c.checkBlockMalformed(blk)
	if err != nil {
		return errors.Wrapf(err, "could not verify block with hash %s", hex.EncodeToString(blk.Header.Hash))
	}

	// 2. Verify each tx - stateful
	for _, merklePl := range blk.Txs {
		tx, ok := merklePl.(transactions.TypeInfo)
		if !ok {
			return errors.New("could not assert tx type as TypeInfo")
		}

		err = checkTXDoubleSpent(tx)
		if err != nil {
			return errors.Wrapf(err, "could not verify block with hash %s", hex.EncodeToString(blk.Header.Hash))
		}
	}

	return nil
}

//VerifyTX will verify whether a transaction is valid by checking:
// - Not been double spent
// In order to check if a transaction has been double spent,
// we need to check that the key-image has not been used before.
// The key-image database is equivalently our utxo database in bitcoin.
// - Not malformed
// Returns nil if a tx is valid
func (c Chain) VerifyTX(tx transactions.TypeInfo) error {
	err := checkTXMalformed(tx)
	if err != nil {
		return err
	}

	err = checkRangeProof(rangeproof.Proof{})
	if err != nil {
		return err
	}

	err = checkTXDoubleSpent(tx)
	if err != nil {
		return err
	}

	return nil
}

// writeBlock is called after all of the checks on the block pass
// returns nil, if write to database was successful
func (c *Chain) writeBlock(blk block.Block) error {
	return nil
}

// hasBlock checks whether the block passed as an
// argument has already been saved into our database
// returns nil, if block does not exist
func (c Chain) checkBlockExists(blk block.Block) error {

	return c.db.View(func(tx database.Tx) error {
		hdr, err := tx.GetBlockHeaderByHash(blk.Header.Hash)
		if hdr != nil {
			return errors.New("chain: block is already present in the database")
		}
		return err
	})

}

// checks whether a block is malformed,
// which in turn checks whether any transactions are malformed
// These are stateless checks
// returns nil, if all checks pass
func (c Chain) checkBlockMalformed(blk block.Block) error {

	// 1. Stateless checks on block

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
	err := blk.SetRoot()
	if err != nil {
		return errors.New("could not calculate merkle tree root")
	}
	if !bytes.Equal(tR, blk.Header.TxRoot) {
		return errors.New("merkle root mismatch")
	}

	// 2. Stateless checks on transactions
	for _, merklePl := range blk.Txs {
		tx, ok := merklePl.(transactions.TypeInfo)
		if !ok {
			return errors.New("could not assert tx type as TypeInfo")
		}
		err := checkTXMalformed(tx)
		if err != nil {
			return err
		}
	}

	return nil
}

// Checks whether a transaction
// is malformed. These are stateless checks
func checkTXMalformed(tx transactions.TypeInfo) error {

	// Version

	// XXX: Checks here have been delayed until we refactor the transaction/blocks structure

	err := checkRangeProof(rangeproof.Proof{})
	if err != nil {
		return err
	}
	return nil
}

func checkRangeProof(p rangeproof.Proof) error {
	ok, err := rangeproof.Verify(p)
	if !ok {
		return errors.New("rangeproof verification failed")
	}
	if err != nil {
		return err
	}
	return nil
}

// checks that the transaction has not been spent by checking
// the database for that key image
// returns nil if item not in database
func checkTXDoubleSpent(tx transactions.TypeInfo) error {

	// 1. Check if keyImage is already in database
	return nil
}
