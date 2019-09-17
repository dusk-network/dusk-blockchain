package chain

import (
	"bytes"
	"encoding/binary"
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
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
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
	k1 := newProvisioner(100, eb, 0, 10)
	k2 := newProvisioner(100, eb, 0, 10)
	k3 := newProvisioner(100, eb, 0, 1)
	keys := []user.Keys{k1, k2, k3}

	// Update round. This should not remove the third provisioner from our committee
	consensus.UpdateRound(eb, 2)

	// Create block 1
	blk := helper.RandomBlock(t, 1, 1)
	// Remove all txs except coinbase, as the helper transactions do not pass verification
	blk.Txs = blk.Txs[0:1]
	blk.SetRoot()
	blk.SetHash()
	// Add cert and prev hash
	blk.Header.Certificate = createMockedCertificate(blk.Header.Hash, 1, keys)
	blk.Header.PrevBlockHash = chain.prevBlock.Header.Hash
	// Accept it
	assert.NoError(t, chain.AcceptBlock(*blk))
	// Provisioner with k3 should no longer be in the committee now
	// assert.False(t, c.IsMember(k3.BLSPubKeyBytes, 2, 1))
}

func newProvisioner(stake uint64, eb *wire.EventBus, startHeight, endHeight uint64) user.Keys {
	k, _ := user.NewRandKeys()
	publishNewStake(stake, eb, startHeight, endHeight, k)
	return k
}

func publishNewStake(stake uint64, eb *wire.EventBus, startHeight, endHeight uint64, k user.Keys) {
	buffer := bytes.NewBuffer(*k.EdPubKey)
	_ = encoding.WriteVarBytes(buffer, k.BLSPubKeyBytes)

	_ = encoding.WriteUint64(buffer, binary.LittleEndian, stake)
	_ = encoding.WriteUint64(buffer, binary.LittleEndian, startHeight)
	_ = encoding.WriteUint64(buffer, binary.LittleEndian, endHeight)

	eb.Publish(msg.NewProvisionerTopic, buffer)
}
