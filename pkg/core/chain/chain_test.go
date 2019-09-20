package chain

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	_ "github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
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

func createMockedCertificate(hash []byte, round uint64, keys []user.Keys, p *user.Provisioners) *block.Certificate {
	votes := agreement.GenVotes(hash, round, 1, keys, p.CreateVotingCommittee(round, 1, len(p.Members)))
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
	chain, err := New(eb, rpc)

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

// Make sure that certificates can still be properly verified when a provisioner is removed on round update.
func TestCertificateExpiredProvisioner(t *testing.T) {
	eb := wire.NewEventBus()
	rpc := wire.NewRPCBus()
	chain, err := New(eb, rpc)

	assert.Nil(t, err)
	defer chain.Close()

	// Add some provisioners to our chain, including one that is just about to expire
	p, k := consensus.MockProvisioners(3)
	p.Members[string(k[0].BLSPubKeyBytes)].Stakes[0].EndHeight = 1

	// Update round. This should not remove the third provisioner from our committee
	eb.Publish(msg.RoundUpdateTopic, consensus.MockRoundUpdateBuffer(2, p, nil))

	// Create block 1
	blk := helper.RandomBlock(t, 1, 1)
	// Remove all txs except coinbase, as the helper transactions do not pass verification
	blk.Txs = blk.Txs[0:1]
	blk.SetRoot()
	blk.SetHash()
	// Add cert and prev hash
	blk.Header.Certificate = createMockedCertificate(blk.Header.Hash, 1, k, p)
	blk.Header.PrevBlockHash = chain.prevBlock.Header.Hash
	// Accept it
	assert.NoError(t, chain.AcceptBlock(*blk))
	// Provisioner with k3 should no longer be in the committee now
	// assert.False(t, c.IsMember(k3.BLSPubKeyBytes, 2, 1))
}
