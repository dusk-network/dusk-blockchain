package chain

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	_ "github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/stretchr/testify/assert"
)

/*
func TestDemoSaveFunctionality(t *testing.T) {

	eb := wire.NewEventBus()
	rpc := wire.NewRPCBus()
	c, keys := agreement.MockCommittee(2, true, 2)
	chain, err := New(eb, rpc, c)

	assert.Nil(t, err)

	defer chain.Close()

	for i := 1; i < 5; i++ {
		nextBlock := helper.RandomBlock(t, 200, 10)
		nextBlock.Header.PrevBlockHash = chain.prevBlock.Header.Hash
		nextBlock.Header.Height = uint64(i)

		// mock certificate to pass test
		cert := createMockedCertificate(nextBlock.Header.Hash, nextBlock.Header.Height, keys)
		nextBlock.Header.Certificate = cert
		err = chain.AcceptBlock(*nextBlock)
		assert.NoError(t, err)
		// Do this to avoid errors with the timestamp when accepting blocks
		time.Sleep(1 * time.Second)
	}

	err = chain.AcceptBlock(chain.prevBlock)
	assert.Error(t, err)

}
*/

func createMockedCertificate(hash []byte, round uint64, keys []user.Keys) *block.Certificate {
	votes := agreement.GenVotes(hash, round, 1, keys)
	return &block.Certificate{
		StepOneBatchedSig: votes[0].Signature.Compress(),
		StepTwoBatchedSig: votes[1].Signature.Compress(),
		Step:              1,
		StepOneCommittee:  votes[0].BitSet,
		StepTwoCommittee:  votes[1].BitSet,
	}
}

func TestFetchTip(t *testing.T) {
	eb := wire.NewEventBus()
	rpc := wire.NewRPCBus()
	chain, err := New(eb, rpc, nil)

	assert.Nil(t, err)
	defer chain.Close()

	// on a modern chain, state(tip) must point at genesis
	var s *database.State
	err = chain.db.View(func(t database.Transaction) error {
		s, err = t.FetchState()
		return err
	})

	assert.Nil(t, err)

	assert.Equal(t, chain.prevBlock.Header.Hash, s.TipHash)
}
