package block_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/block"

	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodeBlock(t *testing.T) {

	assert := assert.New(t)

	// random block
	blk := helper.RandomBlock(t, 200, 2)

	// Encode block into a buffer
	buf := new(bytes.Buffer)
	err := block.Marshal(buf, blk)
	assert.Nil(err)

	// Decode buffer into a block struct
	decBlk := block.NewBlock()
	err = block.Unmarshal(buf, decBlk)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(blk.Equals(decBlk))

}
