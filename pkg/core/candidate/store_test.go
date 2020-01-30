package candidate

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/block"
	"github.com/stretchr/testify/assert"
)

// Test basic storage-related functionality.
func TestStoreFetchClear(t *testing.T) {
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
	eb := eventbus.New()
	rpc := rpcbus.New()
	b := NewBroker(eb, rpc)
	go b.Listen()

	streamer := eventbus.NewStupidStreamer()
	eb.Subscribe(topics.Gossip, eventbus.NewStreamListener(streamer))
	doneChan := make(chan error, 1)

	// Fetch a candidate we don't have
	genesis := config.DecodeGenesis()
	go func() {
		candidateBuf, err := rpc.Call(rpcbus.GetCandidate, rpcbus.Request{*bytes.NewBuffer(genesis.Header.Hash), make(chan rpcbus.Response, 1)}, 0)
		if err != nil {
			doneChan <- err
			return
		}

		c := message.NewCandidate()
		if err := message.UnmarshalCandidate(&candidateBuf, c); err != nil {
			doneChan <- err
			return
		}

		if !assert.True(t, c.Block.Equals(genesis)) {
			var err error
			if !c.Block.Header.Equals(genesis.Header) {
				err = fmt.Errorf("Candidate Header (hash: %s) is not equal to Genesis Header (hash: %s)", c, util.StringifyBytes(genesis.Header.Hash))
			} else if len(c.Block.Txs) != len(genesis.Txs) {
				err = fmt.Errorf("Candidate block (%s) has a different number of txs from Genesis (%s)", c, strconv.Itoa(len(genesis.Txs)))
			} else {
				for i, tx := range genesis.Txs {
					otherTx := c.Block.Txs[i]

					if !tx.Equals(otherTx) {
						err = fmt.Errorf("Candidate Block Tx #%d is different from genesis block Tx", i)
						break
					}
				}
				panic("problem with block.Block.Equals")
			}
			doneChan <- err
			return
		}

		doneChan <- nil
	}()

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
func mockCandidate() message.Candidate {
	genesis := config.DecodeGenesis()
	cert := block.EmptyCertificate()
	return message.MakeCandidate(genesis, cert)
}
