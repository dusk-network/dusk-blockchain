package transactions

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeOutput(t *testing.T) {
	assert := assert.New(t)

	// Random output
	out, err := randomOutput(t, false)
	assert.Nil(err)

	// Encode random output into buffer
	buf := new(bytes.Buffer)
	err = out.Encode(buf)
	assert.Nil(err)

	// Decode buffer into a new output struct
	decOut := &Output{}
	err = decOut.Decode(buf)
	assert.Nil(err)

	// Decoded output should equal original
	assert.True(decOut.Equals(out))
}

func TestMalformedOutput(t *testing.T) {

	// random malformed output
	// should return an error and nil output object
	out, err := randomOutput(t, true)
	assert.Nil(t, out)
	assert.NotNil(t, err)
}

func randomOutput(t *testing.T, malformed bool) (*Output, error) {

	var commSize, keySize, proofSize uint32 = 32, 32, 4500

	if malformed {
		commSize = 45 // This does not have an effect, while Bidding transaction can have clear text
		// and so the commitment size is not fixed
		keySize = 23
	}

	comm := randomSlice(t, commSize)
	key := randomSlice(t, keySize)
	proof := randomSlice(t, proofSize)

	return NewOutput(comm, key, proof)
}

func randomOutputs(t *testing.T, size int, malformed bool) Outputs {

	var outs Outputs

	for i := 0; i < size; i++ {
		out, err := randomOutput(t, malformed)
		if !malformed {
			assert.Nil(t, err)
			assert.NotNil(t, out)
			outs = append(outs, out)
		}
	}

	return outs
}
