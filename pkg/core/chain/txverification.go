package chain

import (
	"github.com/pkg/errors"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

func (c *Chain) checkSpecialFields(tx transactions.Transaction) error {
	switch x := tx.(type) {
	case *transactions.TimeLock:
		return c.verifyTimelock(x)
	case *transactions.Bid:
		return c.verifyBid(x)
	case *transactions.Coinbase:
		return c.verifyCoinbase(x)
	case *transactions.Stake:
		return c.verifyStake(x)
	default:
		return errors.New("unknown transaction type")
	}
}

func (c *Chain) verifyCoinbase(tx *transactions.Coinbase) error {
	return nil
}

func (c *Chain) verifyBid(tx *transactions.Bid) error {
	return nil
}

func (c *Chain) verifyStake(tx *transactions.Stake) error {
	return nil
}

func (c *Chain) verifyTimelock(tx *transactions.TimeLock) error {
	return nil
}
