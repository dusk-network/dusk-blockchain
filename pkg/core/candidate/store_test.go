package candidate

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/stretchr/testify/assert"
)

// Test basic storage-related functionality.
func TestStoreFetchClear(t *testing.T) {
	//t.Skip("feature-419: Genesis block is broken. Unskip after moving to smart contract staking")
	c := newStore()

	// Store a candidate
	candidate := mockCandidate()
	c.storeCandidateMessage(candidate)

	// Fetch it now
	// Hash is genesis hash
	genesis := config.DecodeGenesis()
	fetched := c.fetchCandidateMessage(genesis.Header.Hash)
	assert.NotNil(t, fetched)

	// Correctness checks
	assert.True(t, genesis.Equals(fetched.Block))
	assert.True(t, fetched.Certificate.Equals(block.EmptyCertificate()))

	// Check that Clear empties the entire candidate store
	n := c.Clear(1)
	assert.Equal(t, 1, n)
	assert.Empty(t, c.messages)
}

// Test the candidate request functionality.
func TestRequestCandidate(t *testing.T) {
	// this test uses mockCandidate which decode the broken genesis block
	t.Skip("feature-419: Genesis block is broken. Unskip after moving to smart contract staking")
	eb := eventbus.New()
	rpc := rpcbus.New()
	b := NewBroker(eb, rpc)
	go b.Listen()

	streamer := eventbus.NewStupidStreamer()
	eb.Subscribe(topics.Gossip, eventbus.NewStreamListener(streamer))
	doneChan := make(chan error, 1)

	// Fetch a candidate we don't have
	genesis := config.DecodeGenesis()
	go func(errChan chan<- error) {
		req := rpcbus.NewRequest(*bytes.NewBuffer(genesis.Header.Hash))
		resp, err := rpc.Call(topics.GetCandidate, req, 0)
		if err != nil {
			errChan <- err
			return
		}
		c := resp.(message.Candidate)

		if !assert.True(t, c.Block.Equals(genesis)) {
			if !c.Block.Header.Equals(genesis.Header) {
				errChan <- fmt.Errorf("Candidate Header (hash: %s) is not equal to Genesis Header (hash: %s)", c, util.StringifyBytes(genesis.Header.Hash))
				return
			}

			if len(c.Block.Txs) != len(genesis.Txs) {
				errChan <- fmt.Errorf("Candidate block (%s) has a different number of txs from Genesis (%s)", c, strconv.Itoa(len(genesis.Txs)))
				return
			}

			for i, tx := range genesis.Txs {
				otherTx := c.Block.Txs[i]

				if !tx.Equals(otherTx) {
					errChan <- fmt.Errorf("Candidate Block Tx #%d is different from genesis block Tx", i)
					return
				}
			}
			panic("problem with block.Block.Equals")
		}

		errChan <- nil
	}(doneChan)

	// Make sure we receive a GetCandidate message
	bin, err := streamer.Read()
	if err != nil {
		t.Fatal(err)
	}

	if !assert.Equal(t, topics.GetCandidate, topics.Topic(bin[0])) {
		t.FailNow()
	}

	// Make sure we got a request for the genesis block
	if !assert.True(t, bytes.Equal(bin[1:], genesis.Header.Hash)) {
		t.FailNow()
	}

	// Send genesis back as a Candidate message
	cm := mockCandidate()
	msg := message.New(topics.Candidate, cm)
	eb.Publish(topics.Candidate, msg)
	if err := <-doneChan; err != nil {
		t.Fatal(err)
	}
}

// Mocks a candidate message. It is not in the message package since it uses
// the genesis block as mockup block
//nolint:unused
func mockCandidate() message.Candidate {
	genesis := config.DecodeGenesis()
	cert := block.EmptyCertificate()
	return message.MakeCandidate(genesis, cert)
}
