// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package chain

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config/genesis"
	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	_ "github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/loop"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	assert "github.com/stretchr/testify/require"
)

// TestConcurrentAcceptBlock tests that there is no race condition triggered on
// publishing an AcceptedBlock.
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

	assert.NoError(c.acceptBlock(*blk, true), true)

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
	// genesis := helper.RandomBlock(0, 12)
	return NewDBLoader(db, genesis.Decode())
}

func TestFetchTip(t *testing.T) {
	assert := assert.New(t)
	_, chain := setupChainTest(t, 0)

	// on a modern chain, state(tip) must point at genesis
	var s *database.Registry

	err := chain.db.View(func(t database.Transaction) error {
		var err error
		s, err = t.FetchRegistry()
		return err
	})
	assert.NoError(err)

	assert.Equal(chain.tip.Header.Hash, s.TipHash)
}

func TestSyncProgress(t *testing.T) {
	assert := assert.New(t)
	_, c := setupChainTest(t, 0)

	// SyncProgress should be 0% right now
	resp, err := c.GetSyncProgress(context.Background(), &node.EmptyRequest{})
	assert.NoError(err)

	assert.Equal(resp.Progress, float32(0.0))

	// Change tipHeight and then give the chain a block from far in the future
	c.tip.Header.Height = 50
	blk := helper.RandomBlock(100, 1)
	c.ProcessBlockFromNetwork("", message.New(topics.Block, *blk))

	// SyncProgress should be 50%
	resp, err = c.GetSyncProgress(context.Background(), &node.EmptyRequest{})
	assert.NoError(err)

	assert.Equal(resp.Progress, float32(50.0))
}

func TestFallbackProcedure(t *testing.T) {
	t.Skip()

	// Set up a chain instance with mocking verifiers
	_, chain := setupChainTest(t, 1)

	// Accept block at height 1 with Certificate.Step = 6 (second iteration)
	blk1 := helper.RandomBlock(1, 3)

	cert1 := block.EmptyCertificate()
	cert1.Step = 6
	blk1.Header.Certificate = cert1

	chain.ProcessBlockFromNetwork("", message.New(topics.Block, *blk1))

	b, err := chain.loader.BlockAt(1)
	assert.NoError(t, err)

	// assert block was accepted successfully
	assert.True(t, bytes.Equal(b.Header.Hash, blk1.Header.Hash))

	// Enforce fallback procedure at block height 1
	blk2 := helper.RandomBlock(1, 3)

	cert2 := block.EmptyCertificate()
	cert2.Step = 3 // (first iteration certificate)
	blk2.Header.Certificate = cert2

	chain.ProcessBlockFromNetwork("", message.New(topics.Block, *blk2))

	b, err = chain.loader.BlockAt(1)
	assert.NoError(t, err)

	// assert block at height 1 has been rewritten from fallback procedure
	assert.True(t, bytes.Equal(b.Header.Hash, blk2.Header.Hash))

	// Ensure fallback will be canceled if blk is from higher iteration
	chain.ProcessBlockFromNetwork("", message.New(topics.Block, *blk1))

	b, err = chain.loader.BlockAt(1)
	assert.NoError(t, err)

	assert.True(t, bytes.Equal(b.Header.Hash, blk2.Header.Hash))
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
	blk.Header.StateHash = make([]byte, 32)
	return blk
}

func setupChainTest(t *testing.T, startAtHeight uint64) (*eventbus.EventBus, *Chain) {
	eb := eventbus.New()
	rpc := rpcbus.New()

	_, db := heavy.CreateDBConnection()
	loader := createLoader(db)

	proxy := &transactions.MockProxy{
		E: transactions.MockExecutor(startAtHeight),
	}

	BLSKeys := key.NewRandKeys()

	e := &consensus.Emitter{
		EventBus:    eb,
		RPCBus:      rpc,
		Keys:        BLSKeys,
		TimerLength: 5 * time.Second,
	}

	l := loop.New(e)

	c, err := New(context.Background(), db, eb, rpc, loader, &MockVerifier{}, nil, proxy, l)
	assert.NoError(t, err)

	c.RestartConsensus()
	return eb, c
}
