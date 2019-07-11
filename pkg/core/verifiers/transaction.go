package verifiers

import (
	"bytes"
	"fmt"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/pkg/errors"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/mlsag"
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

	if tx.Fee < uint64(config.MinFee) {
		return errors.New("fee too low")
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
	// rp := rangeproof.Proof{}
	// buf := bytes.NewReader(tx.RangeProof)
	// err := rp.Decode(buf, true)
	// if err != nil {
	// 	return err
	// }

	// var commitments []pedersen.Commitment
	// for _, output := range tx.Outputs {
	// 	var comm pedersen.Commitment

	// 	commBuff := bytes.NewReader(output.Commitment)
	// 	err = comm.Decode(commBuff)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	commitments = append(commitments, comm)
	// }

	// if err := checkRangeProof(rp); err != nil {
	// 	return err
	// }
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
	if lockTime > transactions.MaxLockTime {
		return errors.New("timelock greater than MaxTimeLock")
	}
	return nil
}

func checkRangeProof(p rangeproof.Proof) error {
	_, err := rangeproof.Verify(p)
	return err
}

// checks that the transaction has not been spent by checking the database for that key image
// returns nil if item not in database
func checkTXDoubleSpent(db database.DB, inputs transactions.Inputs) error {

	err := db.View(func(t database.Transaction) error {
		for _, input := range inputs {
			// Decode signature
			var sig mlsag.Signature
			buf := bytes.NewReader(input.Signature)
			err := sig.Decode(buf, true)
			if err != nil {
				return err
			}

			// Check First key in verification is valid
			for _, keyV := range sig.PubKeys {
				key := keyV.OutputKey()
				exists, err := t.FetchOutputExists(key.Bytes())
				if err != nil {
					return err
				}
				if !exists {
					return errors.New("This key is not a previous output ")
				}
			}

		}

		return nil
	})
	if err != nil {
		return err
	}

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
