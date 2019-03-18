package transactions

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeStake(t *testing.T) {

	assert := assert.New(t)

	// random Stake tx
	tx, err := randomStakeTx(t, false)
	assert.Nil(err)

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	err = tx.Encode(buf)
	assert.Nil(err)

	// Decode buffer into a Stake TX struct
	decTX := &Stake{}
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
}

func TestEqualsMethodStake(t *testing.T) {

	assert := assert.New(t)

	a, err := randomStakeTx(t, false)
	assert.Nil(err)
	b, err := randomStakeTx(t, false)
	assert.Nil(err)
	c := a

	assert.False(a.Equals(b))
	assert.False(b.Equals(c))
	assert.True(a.Equals(c))
}

func randomStakeTx(t *testing.T, malformed bool) (*Stake, error) {

	var numInputs, numOutputs = 23, 34
	var lock uint64 = 20000
	var fee uint64 = 20

	edKey := randomSlice(t, 32)
	blsKey := randomSlice(t, 33)

	tx, err := NewStake(0, lock, fee, edKey, blsKey)
	if err != nil {
		return tx, err
	}

	// Inputs
	tx.Inputs = randomInputs(t, numInputs, malformed)

	// Outputs
	tx.Outputs = randomOutputs(t, numOutputs, malformed)

	return tx, nil
}
