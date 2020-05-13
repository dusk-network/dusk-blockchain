package transactions_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	assert "github.com/stretchr/testify/require"
)

// FIXME: 500 - Re-enable these tests and use table testing
func TestEncodeDecodeStandard(t *testing.T) {
	assert := assert.New(t)

	// random standard tx
	tx := transactions.RandTx()

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	err := transactions.Marshal(buf, tx)
	assert.Nil(err)

	// Decode buffer into a standard TX struct
	decTX, err := transactions.Unmarshal(buf)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(transactions.Equal(tx, decTX))

	// Check that Hashes are equal
	txid, err := tx.CalculateHash()
	assert.Nil(err)

	decTxid, err := decTX.CalculateHash()
	assert.Nil(err)

	assert.True(bytes.Equal(txid, decTxid))

	// Check that type is correct
	assert.Equal(transactions.Tx, decTX.Type())
}

func TestEqualsMethodStandard(t *testing.T) {
	assert := assert.New(t)

	a := transactions.RandContractCall()
	b := transactions.RandContractCall()
	c := a

	assert.False(transactions.Equal(a, b))
	assert.False(transactions.Equal(b, c))
	assert.True(transactions.Equal(a, c))
}

func TestEncodeDecodeBid(t *testing.T) {

	assert := assert.New(t)

	// random bid tx
	tx := transactions.RandBidTx(0)

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	assert.Nil(transactions.Marshal(buf, tx))

	// Decode buffer into a bid TX struct
	decTX, err := transactions.Unmarshal(buf)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(transactions.Equal(tx, decTX))

	// Check that type is correct
	assert.Equal(transactions.Bid, decTX.Type())
}

func TestEqualsMethodBid(t *testing.T) {
	assert := assert.New(t)

	a := transactions.RandBidTx(0)
	b := transactions.RandBidTx(0)
	c := a

	assert.False(transactions.Equal(a, b))
	assert.False(transactions.Equal(b, c))
	assert.True(transactions.Equal(a, c))
}

func TestEncodeDecodeStake(t *testing.T) {
	assert := assert.New(t)

	// random Stake tx
	tx := transactions.RandStakeTx(0)

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	assert.NoError(transactions.Marshal(buf, tx))

	// Decode buffer into a Stake TX struct
	decTX, err := transactions.Unmarshal(buf)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(transactions.Equal(tx, decTX))
	// Check that type is correct
	assert.Equal(transactions.Stake, decTX.Type())
}

func TestEqualsMethodStake(t *testing.T) {
	assert := assert.New(t)

	a := transactions.RandStakeTx(0)
	b := transactions.RandStakeTx(0)
	c := a

	assert.False(transactions.Equal(a, b))
	assert.False(transactions.Equal(b, c))
	assert.True(transactions.Equal(a, c))
}

func TestEncodeDecodeCoinbase(t *testing.T) {

	assert := assert.New(t)

	// random coinbase tx
	tx := transactions.RandDistributeTx(0, 35)

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	err := transactions.Marshal(buf, tx)
	assert.Nil(err)

	// Decode buffer into a coinbase TX struct
	decTX, err := transactions.Unmarshal(buf)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(transactions.Equal(tx, decTX))
}

func TestEqualsMethodCoinBase(t *testing.T) {
	assert := assert.New(t)

	a := transactions.RandDistributeTx(0, 35)
	b := transactions.RandDistributeTx(0, 35)
	c := a

	assert.False(transactions.Equal(a, b))
	assert.False(transactions.Equal(b, c))
	assert.True(transactions.Equal(a, c))
}

func TestDecodeTransactions(t *testing.T) {
	assert := assert.New(t)
	txs := transactions.RandContractCalls(2, 0, false)
	r := helper.TxsToBuffer(t, txs)

	decTxs := make([]transactions.ContractCall, len(txs))
	for i := 0; i < len(txs); i++ {
		tx, err := transactions.Unmarshal(r)
		assert.NoError(err)
		decTxs[i] = tx
	}

	assert.Equal(len(txs), len(decTxs))

	for i := range txs {
		tx := txs[i]
		decTx := decTxs[i]
		fmt.Println(decTx)
		assert.True(transactions.Equal(tx, decTx))
	}
}
