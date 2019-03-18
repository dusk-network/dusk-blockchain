package transactions

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeStandard(t *testing.T) {

	assert := assert.New(t)

	// random standard tx
	tx := randomStandardTx(t, false)

	// Encode TX into a buffer
	buf := new(bytes.Buffer)
	err := tx.Encode(buf)
	assert.Nil(err)

	// Decode buffer into a standard TX struct
	decTX := &Standard{}
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
	assert.Equal(StandardType, decTX.TxType)
}

func TestEqualsMethodStandard(t *testing.T) {

	assert := assert.New(t)

	a := randomStandardTx(t, false)
	b := randomStandardTx(t, false)
	c := a

	assert.False(a.Equals(b))
	assert.False(b.Equals(c))
	assert.True(a.Equals(c))
}

func randomStandardTx(t *testing.T, malformed bool) *Standard {

	var numInputs, numOutputs = 10, 10
	var fee uint64 = 20

	tx := NewStandard(0, fee)

	// Inputs
	tx.Inputs = randomInputs(t, numInputs, malformed)

	// Outputs
	tx.Outputs = randomOutputs(t, numOutputs, malformed)

	return tx
}
