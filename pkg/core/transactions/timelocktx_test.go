package transactions

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeTLock(t *testing.T) {

	assert := assert.New(t)

	// random timelock tx
	tx := randomTLockTx(t, false)

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	err := tx.Encode(buf)
	assert.Nil(err)

	// Decode buffer into a TimeLock TX struct
	decTX := &TimeLock{}
	err = decTX.Decode(buf)
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
	assert.Equal(TimelockType, decTX.TxType)
}

func TestEqualsMethodTimeLock(t *testing.T) {

	assert := assert.New(t)

	a := randomTLockTx(t, false)
	b := randomTLockTx(t, false)
	c := a

	assert.False(a.Equals(b))
	assert.False(b.Equals(c))
	assert.True(a.Equals(c))
}

func randomTLockTx(t *testing.T, malformed bool) *TimeLock {

	var numInputs, numOutputs = 23, 34
	var lock uint64 = 20000
	var fee uint64 = 20

	tx := NewTimeLock(0, lock, fee)

	// Inputs
	tx.Inputs = randomInputs(t, numInputs, malformed)

	// Outputs
	tx.Outputs = randomOutputs(t, numOutputs, malformed)

	return tx
}
