package candidate

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test basic storage-related functionality.
func TestStoreFetchClear(t *testing.T) {
	assert := require.New(t)
	c := newStore()

	// Store a candidate
	candidate := mockCandidate()
	c.storeCandidateMessage(candidate)

	// Fetch it now
	// Hash is genesis hash
	genesis := config.DecodeGenesis()
	fetched, _ := c.fetchCandidateMessage(genesis.Header.Hash)
	assert.NotNil(fetched)

	// Correctness checks
	assert.True(genesis.Equals(fetched.Block))
	assert.True(fetched.Certificate.Equals(block.EmptyCertificate()))

	// Check that Clear empties the entire candidate store
	n := c.Clear(1)
	assert.Equal(1, n)
	assert.Empty(c.messages)
}

func TestDeepCopy(t *testing.T) {
	assert := assert.New(t)

	cm := mockCandidate()
	clone := cm.Copy().(message.Candidate)

	assert.True(cm.Block.Equals(clone.Block))
	assert.True(cm.Certificate.Equals(clone.Certificate))
}

// Test the candidate request functionality.
func TestRequestCandidate(t *testing.T) {
	//setup viper timeout
	cwd, err := os.Getwd()
	require.Nil(t, err)

	r, err := config.LoadFromFile(cwd + "/../../../dusk.toml")
	require.Nil(t, err)
	config.Mock(&r)

	assert := assert.New(t)
	require := require.New(t)
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

		buf := new(bytes.Buffer)
		_ = encoding.Write256(buf, genesis.Header.Hash)
		_ = encoding.WriteBool(buf, true)

		req := rpcbus.NewRequest(*buf)
		var resp interface{}
		resp, err = rpc.Call(topics.GetCandidate, req, time.Hour)
		if err != nil {
			errChan <- err
			return
		}
		c := resp.(message.Candidate)

		if !assert.True(c.Block.Equals(genesis)) {
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

				if !transactions.Equal(tx, otherTx) {
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
	require.NoError(err)
	require.Equal(topics.GetCandidate, topics.Topic(bin[0]))

	// Make sure we got a request for the genesis block
	require.True(bytes.Equal(bin[1:33], genesis.Header.Hash))

	// Send genesis back as a Candidate message
	cm := mockCandidate()
	msg := message.New(topics.Candidate, cm)
	errList := eb.Publish(topics.Candidate, msg)
	assert.Empty(errList)

	require.NoError(<-doneChan)
}

// Mocks a candidate message. It is not in the message package since it uses
// the genesis block as mockup block
//nolint:unused
func mockCandidate() message.Candidate {
	genesis := config.DecodeGenesis()
	cert := block.EmptyCertificate()
	return message.MakeCandidate(genesis, cert)
}
