package message_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/stretchr/testify/assert"
)

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
	assert.True(tx.Equals(decTX))

	// Check that Hashes are equal
	txid, err := tx.CalculateHash()
	assert.Nil(err)

	decTxid, err := decTX.CalculateHash()
	assert.Nil(err)

	assert.True(bytes.Equal(txid, decTxid))

	// Check that type is correct
	assert.Equal(transactions.StandardType, decTX.(*transactions.Standard).TxType)
}

func TestEqualsMethodStandard(t *testing.T) {

	assert := assert.New(t)

	a := helper.RandomStandardTx(t, false)
	b := helper.RandomStandardTx(t, false)
	c := a

	assert.False(a.Equals(b))
	assert.False(b.Equals(c))
	assert.True(a.Equals(c))
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
	assert.True(tx.Equals(decTX))

	// Check that Hashes are equal
	txid, err := tx.CalculateHash()
	assert.Nil(err)

	decTxid, err := decTX.CalculateHash()
	assert.Nil(err)

	assert.True(bytes.Equal(txid, decTxid))

	// Check that type is correct
	assert.Equal(transactions.BidType, decTX.(*transactions.Bid).TxType)
}

func TestEqualsMethodBid(t *testing.T) {

	assert := assert.New(t)

	a, err := helper.RandomBidTx(t, false)
	assert.Nil(err)
	b, err := helper.RandomBidTx(t, false)
	assert.Nil(err)
	c := a

	assert.False(a.Equals(b))
	assert.False(b.Equals(c))
	assert.True(a.Equals(c))
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
	assert.True(tx.Equals(decTX))

	// Check that Hashes are equal
	txid, err := tx.CalculateHash()
	assert.Nil(err)

	decTxid, err := decTX.CalculateHash()
	assert.Nil(err)

	assert.True(bytes.Equal(txid, decTxid))

	// Check that type is correct
	assert.Equal(transactions.StakeType, decTX.(*transactions.Stake).TxType)
}

func TestEqualsMethodStake(t *testing.T) {

	assert := assert.New(t)

	a, err := helper.RandomStakeTx(t, false)
	assert.Nil(err)
	b, err := helper.RandomStakeTx(t, false)
	assert.Nil(err)
	c := a

	assert.False(a.Equals(b))
	assert.False(b.Equals(c))
	assert.True(a.Equals(c))
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
	assert.True(tx.Equals(decTX))

	// Check that Hashes are equal
	txid, err := tx.CalculateHash()
	assert.Nil(err)

	decTxid, err := decTX.CalculateHash()
	assert.Nil(err)

	assert.True(bytes.Equal(txid, decTxid))
}

func TestEqualsMethodCoinBase(t *testing.T) {

	assert := assert.New(t)

	a := helper.RandomCoinBaseTx(t, false)
	b := helper.RandomCoinBaseTx(t, false)
	c := a

	assert.False(a.Equals(b))
	assert.False(b.Equals(c))
	assert.True(a.Equals(c))
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
	assert.True(tx.Equals(decTX))

	// Check that Hashes are equal
	txid, err := tx.CalculateHash()
	assert.Nil(err)

	decTxid, err := decTX.CalculateHash()
	assert.Nil(err)

	assert.True(bytes.Equal(txid, decTxid))

	// Check that type is correct
	assert.Equal(transactions.TimelockType, decTX.(*transactions.Timelock).TxType)
}

func TestEqualsMethodTimeLock(t *testing.T) {

	assert := assert.New(t)

	a := helper.RandomTLockTx(t, false)
	b := helper.RandomTLockTx(t, false)
	c := a

	assert.False(a.Equals(b))
	assert.False(b.Equals(c))
	assert.True(a.Equals(c))
}

// Test that the tx type has overridden the standard hash function
func TestBidHashNotStandard(t *testing.T) {

	assert := assert.New(t)

	var txs []transactions.Transaction

	// Possible tye

	bidTx, err := helper.RandomBidTx(t, false)
	assert.Nil(err)
	txs = append(txs, bidTx)

	timeLockTx := helper.RandomTLockTx(t, false)
	txs = append(txs, timeLockTx)

	stakeTx, err := helper.RandomStakeTx(t, false)
	assert.Nil(err)
	txs = append(txs, stakeTx)

	for _, tx := range txs {
		standardHash, txHash := calcTxAndStandardHash(t, tx)
		assert.False(bytes.Equal(standardHash, txHash))
	}
}

// calcTxAndStandardHash calculates the hash for the transaction and
// then the hash for the underlying standardTx. This ensures that the txhash being used,
// is not for the standardTx, unless this is explicitly called.
func calcTxAndStandardHash(t *testing.T, tx transactions.Transaction) ([]byte, []byte) {

	standard := tx.StandardTx()
	standardHash, err := standard.CalculateHash()
	require.Nil(t, err)

	// Clear TxID field, as it will be populated by the underlying Standard struct
	standard.TxID = nil

	txHash, err := tx.CalculateHash()
	assert.Nil(t, err)

	return standardHash, txHash
}

func TestDecodeTransactions(t *testing.T) {
	txs := helper.RandomSliceOfTxs(t, 2)
	r := helper.TxsToBuffer(t, txs)

	decTxs := make([]transactions.Transaction, len(txs))
	for i := 0; i < len(txs); i++ {
		tx, err := message.UnmarshalTx(r)
		assert.Nil(t, err)
		decTxs[i] = tx
	}

	assert.Equal(t, len(txs), len(decTxs))

	for i := range txs {
		tx := txs[i]
		decTx := decTxs[i]
		assert.True(t, tx.Equals(decTx))
	}
}
