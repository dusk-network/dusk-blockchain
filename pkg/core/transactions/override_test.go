package transactions

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Consider refactoring, as functionality is the same for all transactions
// without adding an interface/ introducing unnecessary complexity
// Once all transactions have been implemented, we can iterate all tx types to make sure
// that we have tested against all transactions

// Test that the tx type has overriden the standard
// Hash function
func TestBidHashNotStandard(t *testing.T) {

	assert := assert.New(t)

	// Instantiate all possible transactions

	//BidTx
	bidTx, err := randomBidTx(t, false)
	assert.Nil(err)
	// TimeLockTx
	timeLockTX := randomTLockTx(t, false)

	// Clone all possible transactions

	bidTxClone := bidTx
	timeLockTxClone := timeLockTX

	// Calculate `Standard` hash for each tx

	bidSHash, err := bidTx.Standard.Hash()
	assert.Nil(err)

	timeLockSHash, err := timeLockTX.Standard.Hash()
	assert.Nil(err)

	// Calculate hashes for their types

	bidHash, err := bidTxClone.Hash()
	assert.Nil(err)

	timeLockHash, err := timeLockTxClone.Hash()
	assert.Nil(err)

	// ensure that they do not equal their standard hashes
	assert.False(bytes.Equal(bidSHash, bidHash))

	assert.False(bytes.Equal(timeLockSHash, timeLockHash))
}
