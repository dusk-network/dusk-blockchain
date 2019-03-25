package transactions_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	helper "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

func TestEncodeDecodeOutput(t *testing.T) {
	assert := assert.New(t)

	// Random output
	out, err := helper.RandomOutput(t, false)
	assert.Nil(err)

	// Encode random output into buffer
	buf := new(bytes.Buffer)
	err = out.Encode(buf)
	assert.Nil(err)

	// Decode buffer into a new output struct
	decOut := &transactions.Output{}
	err = decOut.Decode(buf)
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
