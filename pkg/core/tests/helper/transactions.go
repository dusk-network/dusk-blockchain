package helper

import (
	"encoding/binary"
	"testing"

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

	tx, err := transactions.NewBid(0, lockTime, fee, M)
	if err != nil {
		return tx, err
	}

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

	// Do this to pass verification on the chain
	// TODO: this currently doesn't make much sense, fix before release
	bs := make([]byte, 32)
	binary.LittleEndian.PutUint64(bs, config.GeneratorReward)
	tx.Rewards[0].Commitment = bs

	return tx
}

// RandomTLockTx returns a random timelock transaction for testing
func RandomTLockTx(t *testing.T, malformed bool) *transactions.TimeLock {
	var numInputs, numOutputs = 23, 34

	tx := transactions.NewTimeLock(0, lockTime, fee)

	// Inputs
	tx.Inputs = RandomInputs(t, numInputs, malformed)

	// Outputs
	tx.Outputs = RandomOutputs(t, numOutputs, malformed)

	return tx
}

// RandomStandardTx returns a random standard tx for testing
func RandomStandardTx(t *testing.T, malformed bool) *transactions.Standard {
	var numInputs, numOutputs = 10, 10

	tx := transactions.NewStandard(0, fee)

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

	tx, err := transactions.NewStake(0, lockTime, fee, edKey, blsKey)
	if err != nil {
		return tx, err
	}

	// Inputs
	tx.Inputs = RandomInputs(t, numInputs, malformed)

	// Outputs
	tx.Outputs = RandomOutputs(t, numOutputs, malformed)

	return tx, nil
}
