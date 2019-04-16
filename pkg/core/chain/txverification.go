package chain

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof"
)

// AcceptTx will verify whether a transaction is valid by checking:
// - It has not been double spent
// - It is not malformed
// Returns nil if a tx is valid
func (c *Chain) AcceptTx(tx transactions.Transaction) error {

	txID, err := tx.CalculateHash()

	if err != nil {
		return err
	}

	err = c.db.View(func(t database.Transaction) error {
		_, _, _, err := t.FetchBlockTxByHash(txID)
		return err
	})

	// Expect here TxNotFound error to continue
	if err != database.ErrTxNotFound {

		if err == nil {
			err = errors.New("tx already exists")
		}

		return err
	}

	approxBlockTime := uint64(consensusSeconds) + uint64(c.PrevBlock.Header.Timestamp)

	if err := c.verifyTX(0, approxBlockTime, tx); err != nil {
		return err
	}

	if err := c.propagateTx(tx); err != nil {
		return err
	}

	return nil
}

// VerifyTX will verify whether a transaction is valid by checking:
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

func (c *Chain) checkSpecialFields(txIndex uint64, blockTime uint64, tx transactions.Transaction) error {
	switch x := tx.(type) {
	case *transactions.TimeLock:
		return c.verifyTimelock(txIndex, blockTime, x)
	case *transactions.Bid:
		if err := c.verifyBid(txIndex, blockTime, x); err != nil {
			return err
		}

		// TODO: remove this after demo
		// add blind bidder once tx is accepted, and tell the server
		buffer := new(bytes.Buffer)
		if err := x.Encode(buffer); err != nil {
			return err
		}
		c.eventBus.Publish(msg.BidTopic, buffer)
		// Applicable only for the demo
		// c.db.writeTX(x)
		return c.addBidder(x)
	case *transactions.Coinbase:
		return c.verifyCoinbase(txIndex, x)
	case *transactions.Stake:
		if err := c.verifyStake(txIndex, blockTime, x); err != nil {
			return err
		}

		// TODO: remove this after demo
		// add provisioner once tx is accepted, and tell the server
		buffer := new(bytes.Buffer)
		if err := x.Encode(buffer); err != nil {
			return err
		}
		c.eventBus.Publish(msg.StakeTopic, buffer)
		// Applicable only for the demo
		// c.db.writeTX(x)
		return c.addProvisioner(x)
	case *transactions.Standard:
		return c.verifyStandard(x)
	default:
		return errors.New("unknown transaction type")
	}
}

func (c *Chain) verifyStandard(tx *transactions.Standard) error {
	return nil
}

func (c *Chain) verifyCoinbase(txIndex uint64, tx *transactions.Coinbase) error {
	if txIndex != 0 {
		return errors.New("coinbase transaction is not in the first position")
	}
	return nil
}

func (c *Chain) verifyBid(index uint64, blockTime uint64, tx *transactions.Bid) error {
	if err := c.checkLockTimeValid(tx.Lock, blockTime); err != nil {
		return err
	}
	return nil
}

func (c *Chain) verifyStake(index uint64, blockTime uint64, tx *transactions.Stake) error {
	if err := c.checkLockTimeValid(tx.Lock, blockTime); err != nil {
		return err
	}
	return nil
}

func (c *Chain) verifyTimelock(index uint64, blockTime uint64, tx *transactions.TimeLock) error {
	if err := c.checkLockTimeValid(tx.Lock, blockTime); err != nil {
		return err
	}
	return nil
}

func (c *Chain) checkLockTimeValid(lockTime, blockTime uint64) error {
	if lockTime >= transactions.TimeLockBlockZero {
		return c.checkLockValidHeight(lockTime - transactions.TimeLockBlockZero)
	}
	return c.checkLockValidTime(lockTime, blockTime)
}

func (c *Chain) checkLockValidHeight(lockHeight uint64) error {
	nextBlockHeight := c.PrevBlock.Header.Height + 1
	if lockHeight < nextBlockHeight {
		return fmt.Errorf("invalid lock height, lock expired at height %d , it is now height %d", lockHeight, nextBlockHeight)
	}
	return nil
}

func (c *Chain) checkLockValidTime(lockTime, nextBlockTime uint64) error {
	if lockTime < nextBlockTime {
		return fmt.Errorf("invalid lock time, lock expired at time %d , block time is approx. now %d", lockTime, nextBlockTime)
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

	return nil

}
