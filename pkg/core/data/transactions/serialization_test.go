package transactions_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContractCallTxCopy(t *testing.T) {
	assert := require.New(t)
	bid := transactions.RandBidTx(0)
	cpy := bid.ContractTx.Copy()
	assert.True(reflect.DeepEqual(cpy, bid.ContractTx))
	c := bid.Copy().(*transactions.BidTransaction)
	assert.Equal(bid.M, c.M)
	assert.Equal(bid.Commitment, c.Commitment)
	assert.Equal(bid.Pk, c.Pk)
	assert.Equal(bid.R, c.R)
	assert.Equal(bid.Seed, c.Seed)
}

func TestContractCallCopy(t *testing.T) {
	assert := assert.New(t)
	for _, cc := range transactions.RandContractCalls(20, 0, true) {
		if !assert.True(reflect.DeepEqual(cc, cc.Copy())) {
			t.Fatalf("copy is broken for %s", reflect.TypeOf(cc).String())
		}
	}
}

func TestEncodeDecodeStandard(t *testing.T) {
	assert := require.New(t)

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
	assert := require.New(t)

	a := transactions.RandContractCall()
	b := transactions.RandContractCall()
	c := a

	assert.False(transactions.Equal(a, b))
	assert.False(transactions.Equal(b, c))
	assert.True(transactions.Equal(a, c))
}

func TestEncodeDecodeBid(t *testing.T) {

	assert := require.New(t)

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
	assert := require.New(t)

	a := transactions.RandBidTx(0)
	b := transactions.RandBidTx(0)
	c := a

	assert.False(transactions.Equal(a, b))
	assert.False(transactions.Equal(b, c))
	assert.True(transactions.Equal(a, c))
}

func TestEncodeDecodeStake(t *testing.T) {
	assert := require.New(t)

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
	assert := require.New(t)

	a := transactions.RandStakeTx(0)
	b := transactions.RandStakeTx(0)
	c := a

	assert.False(transactions.Equal(a, b))
	assert.False(transactions.Equal(b, c))
	assert.True(transactions.Equal(a, c))
}

func TestEncodeDecodeCoinbase(t *testing.T) {

	assert := require.New(t)

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
	assert := require.New(t)

	a := transactions.RandDistributeTx(0, 35)
	b := transactions.RandDistributeTx(0, 35)
	c := a

	assert.False(transactions.Equal(a, b))
	assert.False(transactions.Equal(b, c))
	assert.True(transactions.Equal(a, c))
}

func TestDecodeTransactions(t *testing.T) {
	assert := require.New(t)
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
		assert.True(transactions.Equal(tx, decTx))
	}
}
