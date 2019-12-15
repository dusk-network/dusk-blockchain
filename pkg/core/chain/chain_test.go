package chain

import (
	"bytes"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	_ "github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-wallet/block"
	"github.com/dusk-network/dusk-wallet/key"
	"github.com/dusk-network/dusk-wallet/transactions"
	"github.com/stretchr/testify/assert"
)

// This test ensures the correct behaviour from the Chain, when
// accepting a block from a peer.
func TestAcceptFromPeer(t *testing.T) {
	eb, _, c := setupChainTest(t, false)
	stopConsensusChan := make(chan bytes.Buffer, 1)
	eb.Subscribe(topics.StopConsensus, eventbus.NewChanListener(stopConsensusChan))

	streamer := eventbus.NewGossipStreamer(protocol.TestNet)
	eb.Subscribe(topics.Gossip, eventbus.NewStreamListener(streamer))
	eb.Register(topics.Gossip, processing.NewGossip(protocol.TestNet))

	// First, test accepting a block when the counter is set to not syncing.
	blk := helper.RandomBlock(t, 1, 1)
	buf := new(bytes.Buffer)
	if err := marshalling.MarshalBlock(buf, blk); err != nil {
		t.Fatal(err)
	}

	assert.NoError(t, c.onAcceptBlock(*buf))

	// Function should return before sending the `StopConsensus` message
	select {
	case <-stopConsensusChan:
		t.Fatal("not supposed to get a StopConsensus message")
	case <-time.After(1 * time.Second):
	}

	// Now, test accepting a block with 1 on the sync counter
	c.counter.StartSyncing(1)

	blk = helper.RandomBlock(t, 1, 1)
	blk.SetPrevBlock(c.prevBlock.Header)
	// Strip all but coinbase tx, to avoid unwanted errors
	blk.Txs = blk.Txs[0:1]
	blk.SetRoot()
	blk.SetHash()
	buf = new(bytes.Buffer)
	if err := marshalling.MarshalBlock(buf, blk); err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := c.onAcceptBlock(*buf); err.Error() != "request timeout" {
			t.Fatal(err)
		}
	}()

	// Should receive a StopConsensus message
	<-stopConsensusChan

	// Discard block gossip
	if _, err := streamer.Read(); err != nil {
		t.Fatal(err)
	}

	// Should get a request for round results for round 2
	m, err := streamer.Read()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, topics.GetRoundResults, streamer.SeenTopics()[1])
	var round uint64
	if err := encoding.ReadUint64LE(bytes.NewBuffer(m), &round); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, uint64(2), round)

}

// This test ensures the correct behaviour when accepting a block
// directly from the consensus.
func TestAcceptIntermediate(t *testing.T) {
	eb, _, c := setupChainTest(t, false)
	go c.Listen()
	intermediateChan := make(chan bytes.Buffer, 1)
	eb.Subscribe(topics.IntermediateBlock, eventbus.NewChanListener(intermediateChan))
	roundUpdateChan := make(chan bytes.Buffer, 1)
	eb.Subscribe(topics.RoundUpdate, eventbus.NewChanListener(roundUpdateChan))

	// Make a 'winning' candidate message
	blk := helper.RandomBlock(t, 2, 1)
	cert := block.EmptyCertificate()
	buf := new(bytes.Buffer)
	if err := marshalling.MarshalBlock(buf, blk); err != nil {
		t.Fatal(err)
	}

	if err := marshalling.MarshalCertificate(buf, cert); err != nil {
		t.Fatal(err)
	}

	// Store it
	eb.Publish(topics.Candidate, buf)

	// Now send a `Certificate` message with this block's hash
	// Make a certificate with a different step, to do a proper equality
	// check later
	cert = block.EmptyCertificate()
	cert.Step = 5

	c.handleCertificateMessage(certMsg{blk.Header.Hash, cert})

	// Should have `blk` as intermediate block now
	assert.True(t, blk.Equals(c.intermediateBlock))

	// lastCertificate should be `cert`
	assert.True(t, cert.Equals(c.lastCertificate))

	// Should have gotten `blk` over topics.IntermediateBlock
	blkBuf := <-intermediateChan
	decodedBlk := block.NewBlock()
	if err := marshalling.UnmarshalBlock(&blkBuf, decodedBlk); err != nil {
		t.Fatal(err)
	}

	assert.True(t, blk.Equals(decodedBlk))

	// Should have gotten a round update with proper info
	ruBuf := <-roundUpdateChan
	ru := &consensus.RoundUpdate{}
	if err := consensus.DecodeRound(&ruBuf, ru); err != nil {
		t.Fatal(err)
	}

	// Should coincide with the new intermediate block
	assert.Equal(t, blk.Header.Height+1, ru.Round)
	assert.Equal(t, blk.Header.Hash, ru.Hash)
	assert.Equal(t, blk.Header.Seed, ru.Seed)
}

// If a candidate block is missing to be set as the next intermediate
// block, the Chain should request it.
func TestRequestMissingCandidate(t *testing.T) {

}

func createMockedCertificate(hash []byte, round uint64, keys []key.ConsensusKeys, p *user.Provisioners) *block.Certificate {
	votes := agreement.GenVotes(hash, round, 3, keys, p)
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

// Make sure that certificates can still be properly verified when a provisioner is removed on round update.
func TestCertificateExpiredProvisioner(t *testing.T) {
	eb := eventbus.New()
	rpc := rpcbus.New()
	counter := chainsync.NewCounter(eb)
	chain, err := New(eb, rpc, counter)
	assert.Nil(t, err)
	defer chain.Close()

	// Add some provisioners to our chain, including one that is just about to expire
	p, k := consensus.MockProvisioners(3)
	p.Members[string(k[0].BLSPubKeyBytes)].Stakes[0].EndHeight = 1

	// Update round. This should not remove the third provisioner from our committee
	eb.Publish(topics.RoundUpdate, consensus.MockRoundUpdateBuffer(2, p, nil))

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
	c, err := New(eb, rpc, nil)
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
	if err := c.addProvisioner(keys.EdPubKeyBytes, keys.BLSPubKeyBytes, 500, 0, 1000); err != nil {
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
		if err := c.addProvisioner(keys.EdPubKeyBytes, keys.BLSPubKeyBytes, 500, 0, 1000); err != nil {
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
	counter := chainsync.NewCounter(eb)
	c, err := New(eb, rpc, counter)
	if err != nil {
		t.Fatal(err)
	}

	if !includeGenesis {
		c.removeExpiredBids(transactions.GenesisExpirationHeight)
		c.removeExpiredProvisioners(transactions.GenesisExpirationHeight)
	}

	return eb, rpc, c
}
