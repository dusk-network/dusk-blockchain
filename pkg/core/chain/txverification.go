package chain

import (
	"fmt"

	"github.com/pkg/errors"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

func (c *Chain) checkSpecialFields(index uint32, blockTime uint64, tx transactions.Transaction) error {
	switch x := tx.(type) {
	case *transactions.TimeLock:
		return c.verifyTimelock(index, blockTime, x)
	case *transactions.Bid:
		return c.verifyBid(index, blockTime, x)
	case *transactions.Coinbase:
		return c.verifyCoinbase(index, x)
	case *transactions.Stake:
		return c.verifyStake(index, blockTime, x)
	default:
		return errors.New("unknown transaction type")
	}
}

func (c *Chain) verifyCoinbase(index uint32, tx *transactions.Coinbase) error {
	if index != 0 {
		return errors.New("coinbase transaction is not in the first position")
	}
	return nil
}

func (c *Chain) verifyBid(index uint32, blockTime uint64, tx *transactions.Bid) error {
	if err := c.checkLockTimeValid(tx.Lock, blockTime); err != nil {
		return err
	}
	return nil
}

func (c *Chain) verifyStake(index uint32, blockTime uint64, tx *transactions.Stake) error {
	if err := c.checkLockTimeValid(tx.Lock, blockTime); err != nil {
		return err
	}
	return nil
}

func (c *Chain) verifyTimelock(index uint32, blockTime uint64, tx *transactions.TimeLock) error {
	if err := c.checkLockTimeValid(tx.Lock, blockTime); err != nil {
		return err
	}
	return nil
}

func (c *Chain) checkLockTimeValid(lockTime, blockTime uint64) error {
	if lockTime >= 0x8000000000000000 {
		return c.checkLockValidHeight(lockTime - 0x8000000000000000)
	}
	return c.checkLockValidTime(lockTime, blockTime)
}

func (c *Chain) checkLockValidHeight(lockHeight uint64) error {
	nextBlockHeight := c.prevBlock.Header.Height + 1
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
