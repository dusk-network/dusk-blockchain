package candidate_test

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/chain/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/block"
	"github.com/dusk-network/dusk-wallet/transactions"
	"github.com/stretchr/testify/assert"
)

// Test that all of the functionality around score/block generation works as intended.
// Note that the proof generator is mocked here, so the actual validity of the data
// is not tested.
func TestGeneration(t *testing.T) {
	bus, rBus := eventbus.New(), rpcbus.New()
	// txBatchCount * 4 will be the amount of non-coinbase transactions in a block.
	// see: helper.RandomSliceOfTxs
	txBatchCount := uint16(2)
	round := uint64(25)
	h := candidate.NewHelper(t, bus, rBus, txBatchCount)
	ru := consensus.MockRoundUpdate(round, nil, nil)
	h.Initialize(ru)

	h.TriggerBlockGeneration()

	// Should receive a Score and Candidate message from the generator
	_ = <-h.ScoreChan
	candidateBuf := <-h.CandidateChan
	// Remove topic byte from candidateBuf
	if _, err := candidateBuf.ReadByte(); err != nil {
		t.Fatal(err)
	}

	c := block.NewBlock()
	if err := marshalling.UnmarshalBlock(&candidateBuf, c); err != nil {
		t.Fatal(err)
	}

	// Check correctness for candidate
	// Note that we skip the score, since that message is mostly mocked.

	// Block height should equal the round
	assert.Equal(t, round, c.Header.Height)

	// First transaction should be coinbase
	if _, ok := c.Txs[0].(*transactions.Coinbase); !ok {
		t.Fatal("first transaction in candidate should be a coinbase")
	}

	// Should contain correct amount of txs
	assert.Equal(t, int((txBatchCount*4)+1), len(c.Txs))
}
