package transactions_test

import (
	"bytes"
	"testing"

	helper "github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeOutput(t *testing.T) {
	assert := assert.New(t)

	// Random output
	out, err := helper.RandomOutput(t, false)
	assert.Nil(err)

	// Encode random output into buffer
	buf := new(bytes.Buffer)
	err = transactions.MarshalOutput(buf, out)
	assert.Nil(err)

	// Decode buffer into a new output struct
	decOut := &transactions.Output{}
	err = transactions.UnmarshalOutput(buf, decOut)
	assert.Nil(err)

	// Decoded output should equal original
	assert.True(decOut.Equals(out))
}

func TestMalformedOutput(t *testing.T) {

	// random malformed output
	// should return an error and nil output object
	out, err := helper.RandomOutput(t, true)
	assert.Nil(t, out)
	assert.NotNil(t, err)
}
