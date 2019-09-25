package chain

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	_ "github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-wallet/block"
	"github.com/dusk-network/dusk-wallet/key"
	"github.com/dusk-network/dusk-wallet/transactions"
	"github.com/stretchr/testify/assert"
)

/*
func TestDemoSaveFunctionality(t *testing.T) {

	eb := eventbus.New()
	rpc := rpcbus.New()
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

func createMockedCertificate(hash []byte, round uint64, keys []key.ConsensusKeys, p *user.Provisioners) *block.Certificate {
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
	eb := eventbus.New()
	rpc := rpcbus.New()
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
	eb := eventbus.New()
	rpc := rpcbus.New()
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

func TestAddAndRemoveBid(t *testing.T) {
	eb := eventbus.New()
	rpc := rpcbus.New()
	c, err := New(eb, rpc)
	if err != nil {
		t.Fatal(err)
	}

	bid := createBid(t)

	c.addBid(bid)
	assert.True(t, c.bidList.Contains(bid))

	c.removeBid(bid)
	assert.False(t, c.bidList.Contains(bid))
}

func TestRemoveExpired(t *testing.T) {
	_, _, c := setupChainTest(t, false)

	for i := 0; i < 10; i++ {
		bid := createBid(t)
		c.addBid(bid)
	}

	// Let's change the end heights alternatingly, to make sure the bidlist removes bids properly
	bl := *c.bidList
	for i, bid := range bl {
		if i%2 == 0 {
			bid.EndHeight = 2000
			bl[i] = bid
		}
	}

	c.bidList = &bl

	// All other bids have their end height at 1000 - so let's remove them
	c.removeExpiredBids(1001)

	assert.Equal(t, 5, len(*c.bidList))

	for _, bid := range *c.bidList {
		assert.Equal(t, uint64(2000), bid.EndHeight)
	}
}

// Add and then a remove a provisioner, to check if removal works properly.
func TestRemove(t *testing.T) {
	_, _, c := setupChainTest(t, false)

	keys, _ := key.NewRandConsensusKeys()
	if err := c.addProvisioner(keys.EdPubKeyBytes, keys.BLSPubKeyBytes, 500, 1000); err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, c.p.GetMember(keys.BLSPubKeyBytes))
	assert.Equal(t, 1, len(c.p.Members))

	if !c.removeProvisioner(keys.BLSPubKeyBytes) {
		t.Fatal("could not remove a member we just added")
	}

	assert.Equal(t, 0, len(c.p.Members))
}

func TestRemoveExpiredProvisioners(t *testing.T) {
	_, _, c := setupChainTest(t, false)

	for i := 0; i < 10; i++ {
		keys, _ := key.NewRandConsensusKeys()
		if err := c.addProvisioner(keys.EdPubKeyBytes, keys.BLSPubKeyBytes, 500, 1000); err != nil {
			t.Fatal(err)
		}
	}

	assert.Equal(t, 10, len(c.p.Members))

	var i int
	for _, p := range c.p.Members {
		if i%2 == 0 {
			p.Stakes[0].EndHeight = 2000
		}
		i++
	}

	c.removeExpiredProvisioners(1001)
	assert.Equal(t, 5, len(c.p.Members))
}

func createBid(t *testing.T) user.Bid {
	b, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	var arr [32]byte
	copy(arr[:], b)
	return user.Bid{arr, arr, 1000}
}

func setupChainTest(t *testing.T, includeGenesis bool) (*eventbus.EventBus, *rpcbus.RPCBus, *Chain) {
	eb := eventbus.New()
	rpc := rpcbus.New()
	c, err := New(eb, rpc)
	if err != nil {
		t.Fatal(err)
	}

	if !includeGenesis {
		c.removeExpiredBids(transactions.GenesisExpirationHeight)
		c.removeExpiredProvisioners(transactions.GenesisExpirationHeight)
	}

	return eb, rpc, c
}
