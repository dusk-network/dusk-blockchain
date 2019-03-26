package chain

import (
	"os"
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
)

func TestVerifyBlockWrong(t *testing.T) {
	blk0 := helper.RandomBlock(t)
	chain := &Chain{
		prevBlock: *blk0,
	}
	blk1 := helper.RandomBlock(t)
	err := chain.VerifyBlock(*blk1)
	assert.NotNil(t, err)
}
func TestVerifyBlockRight(t *testing.T) {
	blk0, blk1 := helper.TwoLinkedBlocks(t)

	db, err := NewDatabase("temp", false)
	assert.Nil(t, err)

	chain := &Chain{
		prevBlock: *blk0,
		db:        db,
	}

	err = chain.VerifyBlock(*blk1)
	assert.Nil(t, err)

	os.RemoveAll("temp")
}
func TestVerifyValidLockTime(t *testing.T) {

	blk := helper.RandomBlock(t)
	blk.Header.Height = 100

	chain := &Chain{
		prevBlock: *blk,
	}

	// Error because lock expires at block zero and we are at block height 100
	err := chain.checkLockTimeValid(transactions.TimeLockBlockZero, 0)
	assert.NotNil(t, err)

	// No error because lock expires at block 101 and we are at block 100
	err = chain.checkLockTimeValid(transactions.TimeLockBlockZero+101, 10)
	assert.Nil(t, err)

	// Error because lock expires at time 3450 and we are at time 9870
	err = chain.checkLockTimeValid(3450, 9870)
	assert.NotNil(t, err)

	// No Error because lock expires at time 1200 and we are at time 0
	err = chain.checkLockTimeValid(1200, 0)
	assert.Nil(t, err)
}

func TestVerifyCoinbase(t *testing.T) {
	coinbase := helper.RandomCoinBaseTx(t, false)

	chain := &Chain{}

	err := chain.verifyCoinbase(0, coinbase)
	assert.Nil(t, err)
	err = chain.verifyCoinbase(1, coinbase)
	assert.NotNil(t, err)
}
