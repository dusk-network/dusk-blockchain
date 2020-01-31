package chain

import (
	"bytes"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	_ "github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-wallet/v2/block"
	"github.com/dusk-network/dusk-wallet/v2/key"
	"github.com/dusk-network/dusk-wallet/v2/transactions"
	"github.com/stretchr/testify/assert"
)

// This test ensures the correct behaviour from the Chain, when
// accepting a block from a peer.
func TestAcceptFromPeer(t *testing.T) {
	eb, _, c := setupChainTest(t, false)
	stopConsensusChan := make(chan message.Message, 1)
	eb.Subscribe(topics.StopConsensus, eventbus.NewChanListener(stopConsensusChan))

	streamer := eventbus.NewGossipStreamer(protocol.TestNet)
	eb.Subscribe(topics.Gossip, eventbus.NewStreamListener(streamer))

	// First, test accepting a block when the counter is set to not syncing.
	blk := helper.RandomBlock(t, 1, 1)
	msg := message.New(topics.AcceptedBlock, *blk)

	assert.NoError(t, c.onAcceptBlock(msg))

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
	root, _ := blk.CalculateRoot()
	blk.Header.TxRoot = root
	hash, _ := blk.CalculateHash()
	blk.Header.Hash = hash

	msg = message.New(topics.AcceptedBlock, *blk)

	go func() {
		if err := c.onAcceptBlock(msg); err.Error() != "request timeout" {
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

	if !assert.Equal(t, topics.GetRoundResults, streamer.SeenTopics()[1]) {
		t.FailNow()
	}

	var round uint64
	if err := encoding.ReadUint64LE(bytes.NewBuffer(m), &round); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, uint64(2), round)
}

// This test ensures the correct behaviour when accepting a block
// directly from the consensus.
func TestAcceptIntermediate(t *testing.T) {
	eb, rpc, c := setupChainTest(t, false)
	go c.Listen()
	intermediateChan := make(chan message.Message, 1)
	eb.Subscribe(topics.IntermediateBlock, eventbus.NewChanListener(intermediateChan))
	roundUpdateChan := make(chan message.Message, 1)
	eb.Subscribe(topics.RoundUpdate, eventbus.NewChanListener(roundUpdateChan))

	// Make a 'winning' candidate message
	blk := helper.RandomBlock(t, 2, 1)
	cert := block.EmptyCertificate()
	provideCandidate(rpc, message.MakeCandidate(blk, cert))

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
	blkMsg := <-intermediateChan
	decodedBlk := blkMsg.Payload().(block.Block)

	assert.True(t, decodedBlk.Equals(blk))

	// Should have gotten a round update with proper info
	ruMsg := <-roundUpdateChan
	ru := ruMsg.Payload().(consensus.RoundUpdate)
	// Should coincide with the new intermediate block
	assert.Equal(t, blk.Header.Height+1, ru.Round)
	assert.Equal(t, blk.Header.Hash, ru.Hash)
	assert.Equal(t, blk.Header.Seed, ru.Seed)
}

func TestReturnOnNilIntermediateBlock(t *testing.T) {
	eb, _, c := setupChainTest(t, false)
	intermediateChan := make(chan message.Message, 1)
	eb.Subscribe(topics.IntermediateBlock, eventbus.NewChanListener(intermediateChan))

	// Make a 'winning' candidate message
	blk := helper.RandomBlock(t, 2, 1)
	cert := block.EmptyCertificate()

	cm := message.MakeCandidate(blk, cert)

	// Store it
	eb.Publish(topics.Candidate, message.New(topics.Candidate, cm))

	// Save current prevBlock
	currPrevBlock := c.prevBlock
	// set intermediate block to nil
	c.intermediateBlock = nil

	// Now pretend we finalized on it
	c.handleCertificateMessage(certMsg{blk.Header.Hash, cert})

	// Ensure everything is still the same
	assert.True(t, currPrevBlock.Equals(&c.prevBlock))
	assert.Nil(t, c.intermediateBlock)
}

func provideCandidate(rpc *rpcbus.RPCBus, cm message.Candidate) {
	c := make(chan rpcbus.Request, 1)
	rpc.Register(rpcbus.GetCandidate, c)
	buf := new(bytes.Buffer)
	if err := message.MarshalCandidate(buf, cm); err != nil {
		panic(err)
	}

	go func() {
		r := <-c
		r.RespChan <- rpcbus.Response{*buf, nil}
	}()
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
	ru := consensus.MockRoundUpdate(2, p, nil)
	msg := message.New(topics.RoundUpdate, ru)
	// Update round. This should not remove the third provisioner from our committee
	eb.Publish(topics.RoundUpdate, msg)

	// Create block 1
	blk := helper.RandomBlock(t, 1, 1)
	// Remove all txs except coinbase, as the helper transactions do not pass verification
	blk.Txs = blk.Txs[0:1]
	root, _ := blk.CalculateRoot()
	blk.Header.TxRoot = root
	hash, _ := blk.CalculateHash()
	blk.Header.Hash = hash
	// Add cert and prev hash
	blk.Header.Certificate = message.MockCertificate(blk.Header.Hash, 1, k, p)
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
