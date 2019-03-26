package chain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
)

func TestDemoSaveFunctionality(t *testing.T) {
	chn, err := New()
	assert.Nil(t, err)

	for i := 1; i < 5; i++ {

		nextBlock := helper.RandomBlock(t)
		nextBlock.Header.PrevBlock = chn.prevBlock.Header.Hash
		nextBlock.Header.Height = uint64(i)
		err = chn.AcceptBlock(*nextBlock)
		assert.Nil(t, err)
	}

	err = chn.AcceptBlock(chn.prevBlock)
	assert.NotNil(t, err)

}
