package chain

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	_ "github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"github.com/sirupsen/logrus"
	assert "github.com/stretchr/testify/require"
)

// TestConcurrentAcceptBlock tests that there is no race condition triggered on
// publishing an AcceptedBlock
func TestConcurrentAcceptBlock(t *testing.T) {
	assert := assert.New(t)
	startingHeight := uint64(1)
	eb, _, c := setupChainTest(t, startingHeight)
	go c.Listen()

	// Run two subscribers expecting acceptedBlock message
	acceptedBlock1Chan := make(chan message.Message, 1)
	acceptedBlock2Chan := make(chan message.Message, 1)
	propagatedHeight := uint64(333)

	// First test that we have a concurrent mutation
	first := eb.Subscribe(topics.AcceptedBlock, eventbus.NewChanListener(acceptedBlock1Chan))
	second := eb.Subscribe(topics.AcceptedBlock, eventbus.NewChanListener(acceptedBlock2Chan))

	// testing that unsafe listeners are prone to mutations
	secondBlock := mutateFirstChan(propagatedHeight, eb, acceptedBlock1Chan, acceptedBlock2Chan)
	assert.NotEqual(secondBlock.Header.Height, propagatedHeight)

	// unsubscribing unsafe listeners
	eb.Unsubscribe(topics.AcceptedBlock, first)
	eb.Unsubscribe(topics.AcceptedBlock, second)

	// Now test that the second Block is unaffected by mutations in the first
	first = eb.Subscribe(topics.AcceptedBlock, eventbus.NewSafeChanListener(acceptedBlock1Chan))
	second = eb.Subscribe(topics.AcceptedBlock, eventbus.NewSafeChanListener(acceptedBlock2Chan))

	// testing that unsafe listeners are prone to mutations
	secondBlock = mutateFirstChan(propagatedHeight, eb, acceptedBlock1Chan, acceptedBlock2Chan)
	assert.Equal(secondBlock.Header.Height, propagatedHeight)

	// unsubscribing unsafe listeners
	eb.Unsubscribe(topics.AcceptedBlock, first)
	eb.Unsubscribe(topics.AcceptedBlock, second)
}

func mutateFirstChan(propagatedHeight uint64, eb eventbus.Publisher, acceptedBlock1Chan, acceptedBlock2Chan chan message.Message) block.Block {
	// Propagate accepted block
	propagatedBlock := helper.RandomBlock(propagatedHeight, 3)

	// shadow copy here as block.Header is a reference
	msg := message.New(topics.AcceptedBlock, *propagatedBlock)
	errList := eb.Publish(topics.AcceptedBlock, msg)
	diagnostics.LogPublishErrors("mutateFirstChan, topics.AcceptedBlock", errList)

	// subscriber_1 collecting propagated block
	blkMsg1 := <-acceptedBlock1Chan
	decodedBlk1 := blkMsg1.Payload().(block.Block)

	// subscriber_1 altering the payload
	decodedBlk1.Header.Height = 999

	// subscriber_2 collecting propagated block
	blkMsg2 := <-acceptedBlock2Chan
	return blkMsg2.Payload().(block.Block)
}

// This test ensures the correct behavior from the Chain, when
// accepting a block from a peer.
func TestAcceptFromPeer(t *testing.T) {
	assert := assert.New(t)
	startingHeight := uint64(1)
	eb, _, c := setupChainTest(t, startingHeight)
	stopConsensusChan := make(chan message.Message, 1)
	eb.Subscribe(topics.StopConsensus, eventbus.NewChanListener(stopConsensusChan))

	streamer := eventbus.NewGossipStreamer(protocol.TestNet)
	eb.Subscribe(topics.Gossip, eventbus.NewStreamListener(streamer))

	// First, test accepting a block when the counter is set to not syncing.
	blk := helper.RandomBlock(startingHeight, 1)
	msg := message.New(topics.AcceptedBlock, *blk)

	assert.NoError(c.onAcceptBlock(msg))

	// Function should return before sending the `StopConsensus` message
	select {
	case <-stopConsensusChan:
		t.Fatal("not supposed to get a StopConsensus message")
	case <-time.After(1 * time.Second):
	}

	// Now, test accepting a block with 1 on the sync counter
	c.counter.StartSyncing(1, "test_peer_addr")

	blk = mockAcceptableBlock(c.tip.Get())
	msg = message.New(topics.AcceptedBlock, *blk)

	errChan := make(chan error, 1)
	go func(chan error) {
		if err := c.onAcceptBlock(msg); err.Error() != "request timeout" {
			errChan <- err
		}
	}(errChan)

	// Should receive a StopConsensus message
	select {
	case err := <-errChan:
		assert.NoError(err)
	case <-stopConsensusChan:
	}

	// the order of received stuff cannot be guaranteed. So we just search for
	// getRoundResult topic. If it hasn't been received the test fails.
	// One message should be the block gossip. The other, the round result
	for i := 0; i < 2; i++ {
		m, err := streamer.Read()
		assert.NoError(err)

		if streamer.SeenTopics()[i] == topics.GetRoundResults {
			var round uint64
			err = encoding.ReadUint64LE(bytes.NewBuffer(m), &round)
			assert.NoError(err)

			assert.Equal(uint64(2), round)
			return
		}
	}

	assert.Fail("expected a round result to be received, but it is not in the ringbuffer")
}

// This test ensures the correct behavior when accepting a block
// directly from the consensus.
func TestAcceptIntermediate(t *testing.T) {
	assert := assert.New(t)
	startingHeight := uint64(2)

	eb, rpc, c := setupChainTest(t, startingHeight)
	go c.Listen()

	intermediateChan := make(chan message.Message, 1)
	eb.Subscribe(topics.IntermediateBlock, eventbus.NewChanListener(intermediateChan))
	roundUpdateChan := make(chan message.Message, 1)
	eb.Subscribe(topics.RoundUpdate, eventbus.NewChanListener(roundUpdateChan))

	// Make a 'winning' candidate message
	blk := helper.RandomBlock(startingHeight, 1)
	cert := block.EmptyCertificate()
	provideCandidate(t, rpc, message.MakeCandidate(blk, cert))

	// Now send a `Certificate` message with this block's hash
	// Make a certificate with a different step, to do a proper equality
	// check later
	cert = block.EmptyCertificate()
	cert.Step = 5

	c.handleCertificateMessage(certMsg{blk.Header.Hash, cert, nil})

	// Should have `blk` as intermediate block now
	assert.True(blk.Equals(c.intermediateBlock))

	// lastCertificate should be `cert`
	assert.True(cert.Equals(c.lastCertificate))

	// Should have gotten `blk` over topics.IntermediateBlock
	blkMsg := <-intermediateChan
	decodedBlk := blkMsg.Payload().(block.Block)

	assert.True(decodedBlk.Equals(blk))

	// Should have gotten a round update with proper info
	ruMsg := <-roundUpdateChan
	ru := ruMsg.Payload().(consensus.RoundUpdate)
	// Should coincide with the new intermediate block
	assert.Equal(blk.Header.Height+1, ru.Round)
	assert.Equal(blk.Header.Hash, ru.Hash)
	assert.Equal(blk.Header.Seed, ru.Seed)
}

func TestReturnOnNilIntermediateBlock(t *testing.T) {
	// suppressing expected warning related to not finding a winning block
	// candidate
	logrus.SetLevel(logrus.ErrorLevel)
	assert := assert.New(t)
	startingHeight := uint64(2)

	eb, _, c := setupChainTest(t, startingHeight)
	intermediateChan := make(chan message.Message, 1)
	eb.Subscribe(topics.IntermediateBlock, eventbus.NewChanListener(intermediateChan))

	// Make a 'winning' candidate message
	blk := helper.RandomBlock(startingHeight, 1)
	cert := block.EmptyCertificate()

	cm := message.MakeCandidate(blk, cert)

	// Store it
	errList := eb.Publish(topics.Candidate, message.New(topics.Candidate, cm))
	assert.Empty(errList)

	// Save current prevBlock
	currPrevBlock := c.tip.Get()
	// set intermediate block to nil
	c.intermediateBlock = nil

	// Now pretend we finalized on it
	c.handleCertificateMessage(certMsg{blk.Header.Hash, cert, nil})

	// Ensure everything is still the same
	prevBlock := c.tip.Get()
	assert.True(currPrevBlock.Equals(&prevBlock))
	assert.Nil(c.intermediateBlock)
}

//nolint:unused
func provideCandidate(t *testing.T, rpc *rpcbus.RPCBus, cm message.Candidate) {
	c := make(chan rpcbus.Request, 1)
	err := rpc.Register(topics.GetCandidate, c)
	assert.NoError(t, err)

	go func() {
		r := <-c
		r.RespChan <- rpcbus.NewResponse(cm, nil)
	}()
}

func createMockedCertificate(hash []byte, round uint64, keys []key.Keys, p *user.Provisioners) *block.Certificate {
	votes := message.GenVotes(hash, round, 3, keys, p)
	return &block.Certificate{
		StepOneBatchedSig: votes[0].Signature.Compress(),
		StepTwoBatchedSig: votes[1].Signature.Compress(),
		Step:              1,
		StepOneCommittee:  votes[0].BitSet,
		StepTwoCommittee:  votes[1].BitSet,
	}
}

func createLoader() *DBLoader {
	_, db := heavy.CreateDBConnection()
	//genesis := cfg.DecodeGenesis()
	genesis := helper.RandomBlock(0, 12)
	return NewDBLoader(db, genesis)
}

func TestFetchTip(t *testing.T) {
	assert := assert.New(t)

	eb := eventbus.New()
	rpc := rpcbus.New()
	loader := createLoader()
	chain, err := New(context.Background(), eb, rpc, nil, loader, &MockVerifier{}, nil, nil)

	assert.NoError(err)

	// on a modern chain, state(tip) must point at genesis
	var s *database.State
	err = loader.db.View(func(t database.Transaction) error {
		s, err = t.FetchState()
		return err
	})

	assert.NoError(err)
	assert.Equal(chain.tip.Get().Header.Hash, s.TipHash)
}

func TestRebuildChain(t *testing.T) {
	eb, rb, c := setupChainTest(t, 0)
	go c.Listen()
	catchClearWalletDatabaseRequest(t, rb)

	// Listen for `StopConsensus` messages
	stopConsensusChan := make(chan message.Message, 1)
	eb.Subscribe(topics.StopConsensus, eventbus.NewChanListener(stopConsensusChan))

	// Add a block so that we have a bit of chain state
	// to check against.
	blk := mockAcceptableBlock(c.tip.Get())

	assert.NoError(t, c.AcceptBlock(context.Background(), *blk))

	// Chain prevBlock should now no longer be genesis
	genesis := c.loader.(*DBLoader).genesis
	//genesis := cfg.DecodeGenesis()
	prevBlock := c.tip.Get()
	assert.False(t, genesis.Equals(&prevBlock))

	p, ks := consensus.MockProvisioners(10)
	c.lastCertificate = createMockedCertificate(c.intermediateBlock.Header.Hash, 2, ks, p)
	c.intermediateBlock = helper.RandomBlock(2, 2)

	// Now, send a request to rebuild the chain
	_, err := c.RebuildChain(context.Background(), &node.EmptyRequest{})
	assert.NoError(t, err)

	// We should be back at the genesis chain state
	prevBlock = c.tip.Get()
	assert.True(t, genesis.Equals(&prevBlock))

	assert.True(t, c.lastCertificate.Equals(block.EmptyCertificate()))
	intermediateBlock, err := mockFirstIntermediateBlock(prevBlock.Header)
	assert.NoError(t, err)
	assert.True(t, c.intermediateBlock.Equals(intermediateBlock))

	// Ensure we got a `StopConsensus` message
	<-stopConsensusChan
}

// mock a block which can be accepted by the chain.
// note that this is only valid for height 1, as the certificate
// is not checked on height 1 (for network bootstrapping)
//nolint
func mockAcceptableBlock(prevBlock block.Block) *block.Block {
	// Create block 1
	blk := helper.RandomBlock(1, 1)
	// Add cert and prev hash
	blk.Header.Certificate = block.EmptyCertificate()
	blk.Header.PrevBlockHash = prevBlock.Header.Hash

	return blk
}

func setupChainTest(t *testing.T, startAtHeight uint64) (*eventbus.EventBus, *rpcbus.RPCBus, *Chain) {
	eb := eventbus.New()
	rpc := rpcbus.New()
	counter := chainsync.NewCounter()

	loader := createLoader()
	proxy := &transactions.MockProxy{
		E: transactions.MockExecutor(startAtHeight),
	}
	var c *Chain
	c, err := New(context.Background(), eb, rpc, counter, loader, &MockVerifier{}, nil, proxy.Executor())
	assert.NoError(t, err)

	return eb, rpc, c
}

func catchClearWalletDatabaseRequest(t *testing.T, rb *rpcbus.RPCBus) {
	c := make(chan rpcbus.Request, 1)
	err := rb.Register(topics.ClearWalletDatabase, c)
	assert.NoError(t, err)

	go func() {
		r := <-c
		r.RespChan <- rpcbus.NewResponse(nil, nil)
	}()
}
