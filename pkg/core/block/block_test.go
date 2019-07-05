package block_test

import (
	"bytes"
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
)

func TestEncodeDecodeBlock(t *testing.T) {

	assert := assert.New(t)

	// random block
	blk := helper.RandomBlock(t, 200, 20)

	// Encode block into a buffer
	buf := new(bytes.Buffer)
	err := blk.Encode(buf)
	assert.Nil(err)

	// Decode buffer into a block struct
	decBlk := block.NewBlock()
	err = decBlk.Decode(buf)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(blk.Equals(decBlk))

}
