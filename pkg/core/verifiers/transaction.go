package verifiers

import (
	"fmt"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/pkg/errors"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof"
)

// CheckTx will verify whether a transaction is valid by checking:
// - It has not been double spent
// - It is not malformed
// Index indicates the position that the transaction is in, in a block
// If it is a solo transaction, this is set to 0
// blockTime indicates what time the transaction will be included in a block
// If it is a solo transaction, the blockTime is calculated by using currentBlockTime+consensusSeconds
// Returns nil if a tx is valid
func CheckTx(db database.DB, index uint64, blockTime uint64, tx transactions.Transaction) error {
	if err := CheckStandardTx(db, tx.StandardTX()); err != nil && tx.Type() != transactions.CoinbaseType {
		return err
	}

	if err := CheckSpecialFields(index, blockTime, tx); err != nil {
		return err
	}

	return nil
}

// CheckStandardTx checks whether the standard fields are correct against the
// passed blockchain db. These checks are both stateless and stateful.
func CheckStandardTx(db database.DB, tx transactions.Standard) error {
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
	if err := checkTXDoubleSpent(db, tx.Inputs); err != nil {
		return err
	}

	return nil
}

// CheckSpecialFields TBD
func CheckSpecialFields(txIndex uint64, blockTime uint64, tx transactions.Transaction) error {
	switch x := tx.(type) {
	case *transactions.TimeLock:
		return VerifyTimelock(txIndex, blockTime, x)
	case *transactions.Bid:
		return VerifyBid(txIndex, blockTime, x)
	case *transactions.Coinbase:
		return VerifyCoinbase(txIndex, x)
	case *transactions.Stake:
		return VerifyStake(txIndex, blockTime, x)
	case *transactions.Standard:
		return VerifyStandard(x)
	default:
		return errors.New("unknown transaction type")
	}
}

func VerifyStandard(tx *transactions.Standard) error {
	return nil
}

func VerifyCoinbase(txIndex uint64, tx *transactions.Coinbase) error {
	if txIndex != 0 {
		return errors.New("coinbase transaction is not in the first position")
	}

	if len(tx.Rewards) != 1 {
		return fmt.Errorf("coinbase transaction must include 1 reward output")
	}

	// Ensure the reward is the fixed one
	rewardScalar := ristretto.Scalar{}
	rewardScalar.UnmarshalBinary(tx.Rewards[0].EncryptedAmount)
	if rewardScalar.BigInt().Uint64() != config.GeneratorReward {
		return fmt.Errorf("coinbase transaction must include a fixed reward of %d", config.GeneratorReward)
	}

	return nil
}

func VerifyBid(index uint64, blockTime uint64, tx *transactions.Bid) error {
	if err := checkLockTimeValid(tx.Lock, blockTime); err != nil {
		return err
	}
	return nil
}

func VerifyStake(index uint64, blockTime uint64, tx *transactions.Stake) error {
	if err := checkLockTimeValid(tx.Lock, blockTime); err != nil {
		return err
	}
	return nil
}

func VerifyTimelock(index uint64, blockTime uint64, tx *transactions.TimeLock) error {
	if err := checkLockTimeValid(tx.Lock, blockTime); err != nil {
		return err
	}
	return nil
}

func checkLockTimeValid(lockTime, blockTime uint64) error {
	if lockTime >= transactions.TimeLockBlockZero {
		return checkLockValidHeight(lockTime - transactions.TimeLockBlockZero)
	}
	return checkLockValidTime(lockTime, blockTime)
}

func checkLockValidHeight(lockHeight uint64) error {
	return nil

	// TODO: @Kev Seems lockHeight is expected here but checkLockTimeValid accepts UnixTime not height
	// nextBlockHeight := PrevBlock.Header.Height + 1
	var nextBlockHeight uint64 = 0
	if lockHeight < nextBlockHeight {
		return fmt.Errorf("invalid lock height, lock expired at height %d , it is now height %d", lockHeight, nextBlockHeight)
	}
	return nil
}

func checkLockValidTime(lockTime, nextBlockTime uint64) error {
	return nil
	if lockTime < nextBlockTime {
		return fmt.Errorf("invalid lock time, lock expired at time %d , block time is approx. now %d", lockTime, nextBlockTime)
	}
	return nil
}

func checkRangeProof(p rangeproof.Proof) error {
	return nil
}

// checks that the transaction has not been spent by checking the database for that key image
// returns nil if item not in database
func checkTXDoubleSpent(db database.DB, inputs transactions.Inputs) error {
	return db.View(func(t database.Transaction) error {
		for _, input := range inputs {
			exists, txID, _ := t.FetchKeyImageExists(input.KeyImage)
			if exists || txID != nil {
				return errors.New("already spent")
			}
		}

		return nil
	})
}
