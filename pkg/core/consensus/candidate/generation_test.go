package candidate_test

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	assert "github.com/stretchr/testify/require"
)

// Test that all of the functionality around score/block generation works as intended.
// Note that the proof generator is mocked here, so the actual validity of the data
// is not tested.
func TestGeneration(t *testing.T) {
	assert := assert.New(t)
	bus, rBus := eventbus.New(), rpcbus.New()
	go func() {
		if err := provideCommittee(rBus); err != nil {
			panic(err)
		}
	}()

	// txBatchCount * 4 will be the amount of non-coinbase transactions in a block.
	// see: helper.RandomSliceOfTxs
	txBatchCount := uint16(2)
	round := uint64(25)
	h := candidate.NewHelper(t, bus, rBus, txBatchCount)
	ru := consensus.MockRoundUpdate(round, nil)
	h.Initialize(ru)

	h.TriggerBlockGeneration()

	// Should receive a Score and Candidate message from the generator
	<-h.ScoreChan
	candidateMsg := <-h.CandidateChan
	c := candidateMsg.Payload().(message.Candidate)

	// Check correctness for candidate
	// Note that we skip the score, since that message is mostly mocked.

	// Block height should equal the round
	assert.Equal(round, c.Header.Height)

	// Last transaction should be coinbase
	if _, ok := c.Txs[len(c.Txs)-1].(*transactions.DistributeTransaction); !ok {
		t.Fatal("last transaction in candidate should be a coinbase")
	}

	// Should contain correct amount of txs
	assert.Equal(int((txBatchCount)+1), len(c.Txs))
}

func provideCommittee(rb *rpcbus.RPCBus) error {
	c := make(chan rpcbus.Request, 1)
	if err := rb.Register(topics.GetLastCommittee, c); err != nil {
		return err
	}

	r := <-c
	com := make([][]byte, 0)
	com = append(com, []byte{1, 2, 3}) //nolint
	r.RespChan <- rpcbus.NewResponse(make([][]byte, 0), nil)
	return nil
}
