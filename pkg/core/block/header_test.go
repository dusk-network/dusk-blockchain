package block_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
)

func TestEncodeDecodeHeader(t *testing.T) {

	assert := assert.New(t)

	// Create a random header
	hdr := helper.RandomHeader(t, 200)
	err := hdr.SetHash()
	assert.Nil(err)

	// Encode header into a buffer
	buf := new(bytes.Buffer)
	err = hdr.Encode(buf)
	assert.Nil(err)

	// Decode buffer into a header struct
	decHdr := &block.Header{}
	err = decHdr.Decode(buf)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(hdr.Equals(decHdr))
}
