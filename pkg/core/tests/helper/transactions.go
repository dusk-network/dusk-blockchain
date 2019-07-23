package helper

import (
	"math/big"
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

const (
	lockTime = uint64(2000000000)
	fee      = uint64(20)
)

// RandomSliceOfTxs returns a random slice of transactions for testing
// Each tx batch represents all 4 non-coinbase tx types
func RandomSliceOfTxs(t *testing.T, txsBatchCount uint16) []transactions.Transaction {
	var txs []transactions.Transaction

	txs = append(txs, RandomCoinBaseTx(t, false))

	var i uint16
	for ; i < txsBatchCount; i++ {

		txs = append(txs, RandomStandardTx(t, false))
		txs = append(txs, RandomTLockTx(t, false))

		stake, err := RandomStakeTx(t, false)
		assert.Nil(t, err)
		txs = append(txs, stake)

		bid, err := RandomBidTx(t, false)
		assert.Nil(t, err)
		txs = append(txs, bid)
	}

	return txs
}

// RandomBidTx returns a random bid transaction for testing
func RandomBidTx(t *testing.T, malformed bool) (*transactions.Bid, error) {
	var numInputs, numOutputs = 23, 34
	var M = RandomSlice(t, 32)

	if malformed {
		M = RandomSlice(t, 12)
	}

	R := RandomSlice(t, 32)

	tx, err := transactions.NewBid(0, lockTime, fee, R, M)
	if err != nil {
		return tx, err
	}
	tx.RangeProof = RandomSlice(t, 200)

	// Inputs
	tx.Inputs = RandomInputs(t, numInputs, malformed)

	// Outputs
	tx.Outputs = RandomOutputs(t, numOutputs, malformed)

	return tx, err
}

// RandomCoinBaseTx returns a random coinbase transaction for testing
func RandomCoinBaseTx(t *testing.T, malformed bool) *transactions.Coinbase {
	proof := RandomSlice(t, 2000)
	score := RandomSlice(t, 32)
	R := RandomSlice(t, 32)

	tx := transactions.NewCoinbase(proof, score, R)
	tx.Rewards = RandomOutputs(t, 1, malformed)
	reward := ristretto.Scalar{}
	reward.SetBigInt(big.NewInt(int64(config.GeneratorReward)))
	tx.Rewards[0].EncryptedAmount = reward.Bytes()
	return tx
}

// RandomTLockTx returns a random timelock transaction for testing
func RandomTLockTx(t *testing.T, malformed bool) *transactions.TimeLock {
	var numInputs, numOutputs = 23, 34

	R := RandomSlice(t, 32)

	tx := transactions.NewTimeLock(0, lockTime, fee, R)
	tx.RangeProof = RandomSlice(t, 200)

	// Inputs
	tx.Inputs = RandomInputs(t, numInputs, malformed)

	// Outputs
	tx.Outputs = RandomOutputs(t, numOutputs, malformed)

	return tx
}

// RandomStandardTx returns a random standard tx for testing
func RandomStandardTx(t *testing.T, malformed bool) *transactions.Standard {
	var numInputs, numOutputs = 10, 10

	R := RandomSlice(t, 32)

	tx := transactions.NewStandard(0, fee, R)
	tx.RangeProof = RandomSlice(t, 200)

	// Inputs
	tx.Inputs = RandomInputs(t, numInputs, malformed)

	// Outputs
	tx.Outputs = RandomOutputs(t, numOutputs, malformed)

	return tx
}

// RandomStakeTx returns a random stake tx for testing
func RandomStakeTx(t *testing.T, malformed bool) (*transactions.Stake, error) {
	var numInputs, numOutputs = 23, 34

	edKey := RandomSlice(t, 32)
	blsKey := RandomSlice(t, 33)
	R := RandomSlice(t, 32)

	tx, err := transactions.NewStake(0, lockTime, fee, R, edKey, blsKey)
	if err != nil {
		return tx, err
	}
	tx.RangeProof = RandomSlice(t, 200)

	// Inputs
	tx.Inputs = RandomInputs(t, numInputs, malformed)

	// Outputs
	tx.Outputs = RandomOutputs(t, numOutputs, malformed)

	return tx, nil
}
