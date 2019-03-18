package transactions

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeInput(t *testing.T) {

	assert := assert.New(t)

	// Random input
	in, err := randomInput(t, false)
	assert.Nil(err)

	// Encode random input into buffer
	buf := new(bytes.Buffer)
	err = in.Encode(buf)
	assert.Nil(err)

	// Decode buffer into a new input struct
	decIn := &Input{}
	err = decIn.Decode(buf)
	assert.Nil(err)

	// Decoded input should equal original
	assert.True(decIn.Equals(in))
}
func TestMalformedInput(t *testing.T) {

	// random malformed input
	// should return an error and nil input object
	in, err := randomInput(t, true)
	assert.Nil(t, in)
	assert.NotNil(t, err)
}

func randomSlice(t *testing.T, size uint32) []byte {
	randSlice := make([]byte, size)
	_, err := rand.Read(randSlice)
	assert.Nil(t, err)
	return randSlice
}

func randomInput(t *testing.T, malformed bool) (*Input, error) {

	var kiSize, txidSize, sigSize uint32 = 32, 32, 3400

	if malformed {
		kiSize = 45
		txidSize = 23
	}

	keyImage := randomSlice(t, kiSize)
	txid := randomSlice(t, txidSize)
	sig := randomSlice(t, sigSize)

	return NewInput(keyImage, txid, 2, sig)
}

func randomInputs(t *testing.T, size int, malformed bool) Inputs {

	var ins Inputs

	for i := 0; i < size; i++ {
		in, err := randomInput(t, malformed)
		if !malformed {
			assert.Nil(t, err)
			assert.NotNil(t, in)
			ins = append(ins, in)
		}
	}

	return ins
}
