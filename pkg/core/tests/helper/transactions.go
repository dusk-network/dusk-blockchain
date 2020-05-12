package helper

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
)

//FIXME: 499 - These mocks are the basis of many different tests. They all need
//to be reinstated (and moved to transactions/fixtures.go)

//const (
//	lockTime = uint64(2000000000)
//	fee      = int64(200)
//)
//
//var numInputs, numOutputs = 23, 16

// RandomSliceOfTxs returns a random slice of transactions for testing
// Each tx batch represents all 4 non-coinbase tx types
func RandomSliceOfTxs(t *testing.T, txsBatchCount uint16) []transactions.ContractCall {
	/*
		var txs []transactions.ContractCall

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
	*/
	return nil
}

// RandomBidTx returns a random bid transaction for testing
func RandomBidTx(t *testing.T, malformed bool) (*transactions.BidTransaction, error) {
	/*
		var M = RandomSlice(t, 32)

		if malformed {
			M = RandomSlice(t, 12)
		}

		tx, err := transactions.NewBid(0, 2, fee, lockTime, M)
		if err != nil {
			return tx, err
		}
		rp := randomRangeProofBuffer(t)
		_ = tx.RangeProof.Decode(rp, true)

		// Inputs
		tx.Inputs = RandomInputs(t, numInputs)

		// Outputs
		tx.Outputs = RandomOutputs(t, numOutputs)

		// Set TxID
		hash, err := tx.CalculateHash()
		if err != nil {
			t.Fatal(err)
		}

		tx.TxID = hash

		return tx, err
	*/
	return nil, nil
}

// RandomCoinBaseTx returns a random coinbase transaction for testing
func RandomCoinBaseTx(t *testing.T, malformed bool) *transactions.DistributeTransaction {
	/*
		proof := RandomSlice(t, 2000)
		score := RandomSlice(t, 32)

		tx := transactions.NewCoinbase(proof, score, 2)
		tx.Rewards = RandomOutputs(t, 1)
		tx.Rewards[0].EncryptedAmount.SetBigInt(big.NewInt(int64(config.GeneratorReward)))

		// Set TxID
		hash, err := tx.CalculateHash()
		if err != nil {
			t.Fatal(err)
		}

		tx.TxID = hash
		return tx
	*/
	return nil
}

// RandomTLockTx returns a random timelock transaction for testing
func RandomTLockTx(t *testing.T, malformed bool) transactions.ContractCall {
	/*
		tx, err := transactions.NewTimelock(0, 2, fee, lockTime)
		if err != nil {
			t.Fatal(err)
		}
		rp := randomRangeProofBuffer(t)
		_ = tx.RangeProof.Decode(rp, true)

		// Inputs
		tx.Inputs = RandomInputs(t, numInputs)

		// Outputs
		tx.Outputs = RandomOutputs(t, numOutputs)

		// Set TxID
		hash, err := tx.CalculateHash()
		if err != nil {
			t.Fatal(err)
		}

		tx.TxID = hash

		return tx
	*/
	return nil
}

// RandomStakeTx returns a random stake tx for testing
func RandomStakeTx(t *testing.T, malformed bool) (*transactions.StakeTransaction, error) {
	/*
		blsKey := RandomSlice(t, 33)

		tx, err := transactions.NewStake(0, 2, fee, lockTime, blsKey)
		if err != nil {
			return tx, err
		}
		rp := randomRangeProofBuffer(t)
		_ = tx.RangeProof.Decode(rp, true)

		// Inputs
		tx.Inputs = RandomInputs(t, numInputs)

		// Outputs
		tx.Outputs = RandomOutputs(t, numOutputs)

		// Set TxID
		hash, err := tx.CalculateHash()
		if err != nil {
			t.Fatal(err)
		}

		tx.TxID = hash

		return tx, nil
	*/
	return nil, nil
}

// FixedStandardTx generates an encodable standard Tx with 1 input and 1 output
// It guarantees that for one seed the same standard Tx (incl. TxID) is
// always generated.
func FixedStandardTx(t *testing.T, seed uint64) transactions.ContractCall {
	/*
		seedScalar := ristretto.Scalar{}
		seedScalar.SetBigInt(big.NewInt(0).SetUint64(seed))

		seedPoint := ristretto.Point{}
		seedPoint.ScalarMultBase(&seedScalar)

		netPrefix := byte(2)
		tx, err := transactions.NewStandard(0, netPrefix, int64(seed))
		if err != nil {
			t.Fatal(err)
		}

		tx.R = seedPoint
		tx.RangeProof = fixedRangeProof(t)

		// Add a fixed output
		walletKeys := key.NewKeyPair([]byte{5, 0, 0})
		output := transactions.NewOutput(seedScalar, seedScalar, 0, *walletKeys.PublicKey())
		tx.Outputs = append(tx.Outputs, output)

		privKey, _ := walletKeys.PrivateSpend()
		in := transactions.NewInput(seedScalar, seedScalar, ristretto.Scalar(*privKey))

		sigBuf := fixedSignatureBuffer(t)
		in.Signature = &mlsag.Signature{}
		if err := in.Signature.Decode(sigBuf, true); err != nil {
			t.Fatal(err)
		}

		in.KeyImage = seedPoint
		tx.Inputs = append(tx.Inputs, in)

		return tx
	*/
	return nil
}
