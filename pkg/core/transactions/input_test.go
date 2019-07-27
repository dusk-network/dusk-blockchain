package transactions_test

import (
	"bytes"
	"testing"

	helper "github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeInput(t *testing.T) {

	assert := assert.New(t)

	// Random input
	in, err := helper.RandomInput(t, false)
	assert.Nil(err)

	// Encode random input into buffer
	buf := new(bytes.Buffer)
	err = in.Encode(buf)
	assert.Nil(err)

	// Decode buffer into a new input struct
	decIn := &transactions.Input{}
	err = decIn.Decode(buf)
	assert.Nil(err)

	// Decoded input should equal original
	assert.True(decIn.Equals(in))
}
func TestMalformedInput(t *testing.T) {

	// random malformed input
	// should return an error and nil input object
	in, err := helper.RandomInput(t, true)
	assert.Nil(t, in)
	assert.NotNil(t, err)
}
