package chain

import (
	"context"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	assert "github.com/stretchr/testify/require"
)

func TestSuccessiveBlocks(t *testing.T) {
	assert := assert.New(t)
	s, c, _ := setupSynchronizerTest()

	// tipHeight will be 0, so make the successive block
	blk := helper.RandomBlock(1, 1)
	_, err := s.ProcessBlock(message.New(topics.Block, *blk))
	assert.NoError(err)

	// Block will come through the catchblock channel
	rBlk := <-c
	assert.True(blk.Equals(&rBlk.Blk))
}

func TestFutureBlocks(t *testing.T) {
	assert := assert.New(t)
	s, _, _ := setupSynchronizerTest()

	height := uint64(10)
	blk := helper.RandomBlock(height, 1)
	resp, err := s.ProcessBlock(message.New(topics.Block, *blk))
	assert.NoError(err)

	// Response should be of the GetBlocks topic
	assert.Equal(resp[0].Bytes()[0], uint8(topics.GetBlocks))

	// Block should be in the sequencer
	assert.NotEmpty(s.sequencer.blockPool[height])

	assert.Equal(s.highestSeen, height)
	assert.True(s.syncing)
}

func TestSyncProgress(t *testing.T) {
	assert := assert.New(t)
	s, _, m := setupSynchronizerTest()

	// SyncProgress should be 0% right now
	resp, err := s.GetSyncProgress(context.Background(), &node.EmptyRequest{})
	assert.NoError(err)

	assert.Equal(resp.Progress, float32(0.0))

	// Change tipHeight and then give the synchronizer a block from far in the future
	m.tipHeight = 50
	blk := helper.RandomBlock(100, 1)
	s.ProcessBlock(message.New(topics.Block, *blk))

	// SyncProgress should be 50%
	resp, err = s.GetSyncProgress(context.Background(), &node.EmptyRequest{})
	assert.NoError(err)

	assert.Equal(resp.Progress, float32(50.0))
}

func setupSynchronizerTest() (*Synchronizer, chan consensus.Results, *mockChain) {
	ctx := context.Background()
	eb, rb := eventbus.New(), rpcbus.New()
	catchBlockChan := make(chan consensus.Results, 1)
	m := &mockChain{tipHeight: 0, catchBlockChan: catchBlockChan}

	return NewSynchronizer(ctx, eb, rb, m), catchBlockChan, m
}

type mockChain struct {
	tipHeight      uint64
	catchBlockChan chan consensus.Results
}

func (m *mockChain) CurrentHeight() uint64 {
	return m.tipHeight
}

func (m *mockChain) ProcessSucceedingBlock(blk block.Block) {
	m.catchBlockChan <- consensus.Results{Blk: blk, Err: nil}
}

func (m *mockChain) ProcessSyncBlock(ctx context.Context, blk block.Block) error {
	return nil
}

func (m *mockChain) CrunchBlocks(ctx context.Context) error {
	return nil
}
