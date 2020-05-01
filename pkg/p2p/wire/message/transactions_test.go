package message_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/stretchr/testify/assert"
)

// TODO: these can absolutely be table tests

func TestEncodeDecodeStandard(t *testing.T) {

	assert := assert.New(t)

	// random standard tx
	tx := helper.RandomStandardTx(t, false)

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	err := message.MarshalTx(buf, tx)
	assert.Nil(err)

	// Decode buffer into a standard TX struct
	decTX, err := message.UnmarshalTx(buf)
	assert.Nil(err)

	// Check both structs are equal
	// assert.True(tx.Equals(decTX))

	// Check that Hashes are equal
	txid, err := tx.CalculateHash()
	assert.Nil(err)

	decTxid, err := decTX.CalculateHash()
	assert.Nil(err)

	assert.True(bytes.Equal(txid, decTxid))

	// Check that type is correct
	// assert.Equal(transactions.StandardType, decTX.(*transactions.Standard).TxType)
}

func TestEqualsMethodStandard(t *testing.T) {
	// TODO: add equals method for RUSK/phoenix transactions

	/*
		assert := assert.New(t)

		a := helper.RandomStandardTx(t, false)
		b := helper.RandomStandardTx(t, false)
		c := a

		assert.False(a.Equals(b))
		assert.False(b.Equals(c))
		assert.True(a.Equals(c))
	*/
}

func TestEncodeDecodeBid(t *testing.T) {

	assert := assert.New(t)

	// random bid tx
	tx, err := helper.RandomBidTx(t, false)
	assert.Nil(err)

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	err = message.MarshalTx(buf, tx)
	assert.Nil(err)

	// Decode buffer into a bid TX struct
	decTX, err := message.UnmarshalTx(buf)
	assert.Nil(err)

	// Check both structs are equal
	// TODO: add equals method for RUSK/phoenix transactions
	// assert.True(tx.Equals(decTX))

	assert.True(transactions.Equal(tx, decTX))

	// Check that type is correct
	// assert.Equal(transactions.BidType, decTX.(*transactions.Bid).TxType)
}

func TestEqualsMethodBid(t *testing.T) {
	assert := assert.New(t)

	a, err := helper.RandomBidTx(t, false)
	assert.Nil(err)
	b, err := helper.RandomBidTx(t, false)
	assert.Nil(err)
	c := a

	assert.False(transactions.Equal(a, b))
	assert.False(transactions.Equal(b, c))
	assert.True(transactions.Equal(a, c))
}

func TestEncodeDecodeStake(t *testing.T) {

	assert := assert.New(t)

	// random Stake tx
	tx, err := helper.RandomStakeTx(t, false)
	assert.Nil(err)

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	err = message.MarshalTx(buf, tx)
	assert.Nil(err)

	// Decode buffer into a Stake TX struct
	decTX, err := message.UnmarshalTx(buf)
	assert.Nil(err)

	// Check both structs are equal
	// TODO: add equals method for RUSK/phoenix transactions
	assert.True(transactions.Equal(tx, decTX))
	// Check that type is correct
	// assert.Equal(transactions.StakeType, decTX.(*transactions.Stake).TxType)
}

func TestEqualsMethodStake(t *testing.T) {
	assert := assert.New(t)

	a, err := helper.RandomStakeTx(t, false)
	assert.Nil(err)
	b, err := helper.RandomStakeTx(t, false)
	assert.Nil(err)
	c := a

	assert.False(transactions.Equal(a, b))
	assert.False(transactions.Equal(b, c))
	assert.True(transactions.Equal(a, c))
}

func TestEncodeDecodeCoinbase(t *testing.T) {

	assert := assert.New(t)

	// random coinbase tx
	tx := helper.RandomCoinBaseTx(t, false)

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	err := message.MarshalTx(buf, tx)
	assert.Nil(err)

	// Decode buffer into a coinbase TX struct
	decTX, err := message.UnmarshalTx(buf)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(transactions.Equal(tx, decTX))
}

func TestEqualsMethodCoinBase(t *testing.T) {
	assert := assert.New(t)

	a := helper.RandomCoinBaseTx(t, false)
	b := helper.RandomCoinBaseTx(t, false)
	c := a

	assert.False(transactions.Equal(a, b))
	assert.False(transactions.Equal(b, c))
	assert.True(transactions.Equal(a, c))
}

func TestEncodeDecodeTLock(t *testing.T) {

	assert := assert.New(t)

	// random timelock tx
	tx := helper.RandomTLockTx(t, false)

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	err := message.MarshalTx(buf, tx)
	assert.Nil(err)

	// Decode buffer into a TimeLock TX struct
	decTX, err := message.UnmarshalTx(buf)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(transactions.Equal(tx, decTX))

	// Check that type is correct
	// TODO: this type does not exist in the RUSK migration
	// assert.Equal(transactions.TimelockType, decTX.(*transactions.Timelock).TxType)
}

func TestEqualsMethodTimeLock(t *testing.T) {
	// TODO: add equals method for RUSK/phoenix transactions
	assert := assert.New(t)

	a := helper.RandomTLockTx(t, false)
	b := helper.RandomTLockTx(t, false)
	c := a

	assert.False(transactions.Equal(a, b))
	assert.False(transactions.Equal(b, c))
	assert.True(transactions.Equal(a, c))
}

func TestDecodeTransactions(t *testing.T) {
	txs := helper.RandomSliceOfTxs(t, 2)
	r := helper.TxsToBuffer(t, txs)

	decTxs := make([]transactions.ContractCall, len(txs))
	for i := 0; i < len(txs); i++ {
		tx, err := message.UnmarshalTx(r)
		assert.Nil(t, err)
		decTxs[i] = tx
	}

	assert.Equal(t, len(txs), len(decTxs))

	for i := range txs {
		tx := txs[i]
		decTx := decTxs[i]
		assert.True(t, transactions.Equal(tx, decTx))
	}
}
