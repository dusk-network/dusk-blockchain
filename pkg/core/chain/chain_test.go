package chain

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"
	crypto "github.com/dusk-network/dusk-crypto/hash"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/common"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	_ "github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/loop"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/sirupsen/logrus"
	assert "github.com/stretchr/testify/require"
)

// TestConcurrentAcceptBlock tests that there is no race condition triggered on
// publishing an AcceptedBlock
func TestConcurrentAcceptBlock(t *testing.T) {
	assert := assert.New(t)
	startingHeight := uint64(1)
	eb, _ := setupChainTest(t, startingHeight)

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
	logrus.SetLevel(logrus.InfoLevel)
	assert := assert.New(t)
	startingHeight := uint64(1)
	eb, c := setupChainTest(t, startingHeight)

	d, _ := crypto.RandEntropy(32)
	k, _ := crypto.RandEntropy(32)
	if err := c.db.Update(func(t database.Transaction) error {
		return t.StoreBidValues(d, k, 0, 100000)
	}); err != nil {
		t.Fatal(err)
	}

	streamer := eventbus.NewGossipStreamer(protocol.TestNet)
	eb.Subscribe(topics.Gossip, eventbus.NewStreamListener(streamer))

	blk := mockAcceptableBlock(*c.tip)

	c.ProcessSucceedingBlock(*blk)

	// the order of received stuff cannot be guaranteed. So we just search for
	// getRoundResult topic. If it hasn't been received the test fails.
	// One message should be the block gossip. The other, the round result
	for i := 0; i < 2; i++ {
		m, err := streamer.Read()
		assert.NoError(err)

		if streamer.SeenTopics()[i] == topics.Inv {

			// Read hash of the advertised block
			var decoder message.Inv
			decoder.Decode(bytes.NewBuffer(m))

			assert.Equal(decoder.InvList[0].Type, message.InvTypeBlock)
			assert.True(bytes.Equal(decoder.InvList[0].Hash, blk.Header.Hash))
			return
		}
	}

	assert.Fail("expected a round result to be received, but it is not in the ringbuffer")
}

// This test ensures the correct behavior when accepting a block
// directly from the consensus.
func TestAcceptBlock(t *testing.T) {
	assert := assert.New(t)
	startingHeight := uint64(1)

	eb, c := setupChainTest(t, startingHeight)

	acceptedBlockChan := make(chan message.Message, 1)
	eb.Subscribe(topics.AcceptedBlock, eventbus.NewChanListener(acceptedBlockChan))

	// Make a 'winning' candidate message
	blk := helper.RandomBlock(startingHeight, 1)

	// Now send a `Certificate` message with this block's hash
	// Make a certificate with a different step, to do a proper equality
	// check later
	cert := block.EmptyCertificate()
	cert.Step = 5
	blk.Header.Certificate = cert

	assert.NoError(c.AcceptBlock(c.ctx, *blk))

	// Should have `blk` as blockchain head now
	assert.True(bytes.Equal(blk.Header.Hash, c.tip.Header.Hash))

	// lastCertificate should be `cert`
	assert.True(cert.Equals(c.tip.Header.Certificate))

	// Should have gotten `blk` over topics.AcceptBlock
	blkMsg := <-acceptedBlockChan
	decodedBlk := blkMsg.Payload().(block.Block)

	assert.True(decodedBlk.Equals(c.tip))
}

func createLoader(db database.DB) *DBLoader {
	//genesis := cfg.DecodeGenesis()
	genesis := helper.RandomBlock(0, 12)
	return NewDBLoader(db, genesis)
}

func TestFetchTip(t *testing.T) {
	assert := assert.New(t)
	_, chain := setupChainTest(t, 0)

	// on a modern chain, state(tip) must point at genesis
	var s *database.State
	err := chain.db.View(func(t database.Transaction) error {
		var err error
		s, err = t.FetchState()
		return err
	})

	assert.NoError(err)
	assert.Equal(chain.tip.Header.Hash, s.TipHash)
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

func setupChainTest(t *testing.T, startAtHeight uint64) (*eventbus.EventBus, *Chain) {
	eb := eventbus.New()
	rpc := rpcbus.New()

	_, db := heavy.CreateDBConnection()
	loader := createLoader(db)

	proxy := &transactions.MockProxy{
		E:  transactions.MockExecutor(startAtHeight),
		BG: transactions.MockBlockGenerator{},
	}

	BLSKeys, _ := key.NewRandKeys()
	pk := keys.PublicKey{
		AG: &common.JubJubCompressed{Data: make([]byte, 32)},
		BG: &common.JubJubCompressed{Data: make([]byte, 32)},
	}

	e := &consensus.Emitter{
		EventBus:    eb,
		RPCBus:      rpc,
		Keys:        BLSKeys,
		Proxy:       proxy,
		TimerLength: 5 * time.Second,
	}
	l := loop.New(e, &pk)

	c, err := New(context.Background(), db, eb, rpc, loader, &MockVerifier{}, nil, proxy, l)
	assert.NoError(t, err)
	assert.NoError(t, db.Update(func(tx database.Transaction) error {
		return tx.StoreBidValues(make([]byte, 32), make([]byte, 32), 0, 100000)
	}))

	go c.CrunchBlocks(context.Background())

	return eb, c
}
