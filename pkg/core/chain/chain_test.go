package chain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func TestDemoSaveFunctionality(t *testing.T) {
	eb := wire.New()
	chn, err := New(eb, "testdb")
	assert.Nil(t, err)

	for i := 1; i < 5; i++ {

		nextBlock := helper.RandomBlock(t, 200, 10)
		nextBlock.Header.PrevBlock = chn.PrevBlock.Header.Hash
		nextBlock.Header.Height = uint64(i)
		err = chn.AcceptBlock(*nextBlock)
		assert.Nil(t, err)
	}

	err = chn.AcceptBlock(chn.PrevBlock)
	assert.NotNil(t, err)

}
