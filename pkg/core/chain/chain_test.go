package chain

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/heavy"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func mockConfig(t *testing.T) func() {

	storeDir, err := ioutil.TempDir(os.TempDir(), "chain_test")
	if err != nil {
		t.Fatal(err.Error())
	}

	r := cfg.Registry{}
	r.Database.Dir = storeDir
	r.Database.Driver = heavy.DriverName
	r.General.Network = "testnet"
	cfg.Mock(&r)

	return func() {
		os.RemoveAll(storeDir)
	}
}

func TestDemoSaveFunctionality(t *testing.T) {

	fn := mockConfig(t)
	defer fn()

	eb := wire.NewEventBus()
	chain, err := New(eb)

	assert.Nil(t, err)

	defer chain.Close()

	for i := 1; i < 5; i++ {

		nextBlock := helper.RandomBlock(t, 200, 10)
		nextBlock.Header.PrevBlock = chain.PrevBlock.Header.Hash
		nextBlock.Header.Height = uint64(i)
		err = chain.AcceptBlock(*nextBlock)
		assert.Nil(t, err)
	}

	err = chain.AcceptBlock(chain.PrevBlock)
	assert.NotNil(t, err)

}
