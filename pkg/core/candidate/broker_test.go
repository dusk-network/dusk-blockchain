package candidate_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	assert "github.com/stretchr/testify/require"
)

// Ensures that candidate blocks are only let through if
// a `ValidCandidateHash` message for that block was seen.
func TestValidHashes(t *testing.T) {
	assert := assert.New(t)
	eb, rb := eventbus.New(), rpcbus.New()
	b := candidate.NewBroker(eb, rb)
	go b.Listen()

	// Store two blocks in the queue
	blk := helper.RandomBlock(1, 3)
	blk2 := helper.RandomBlock(1, 3)
	cert := block.EmptyCertificate()
	hash, _ := blk.CalculateHash()
	blk.Header.Hash = hash

	// mocking a header with consistent hash with the block
	hdr := header.Mock()
	hdr.BlockHash = blk.Header.Hash

	cm := message.MakeCandidate(blk, cert)
	msg := message.New(topics.Candidate, cm)
	errList := eb.Publish(topics.Candidate, msg)
	assert.Empty(errList)

	cm2 := message.MakeCandidate(blk2, cert)
	msg2 := message.New(topics.Candidate, cm2)
	errList = eb.Publish(topics.Candidate, msg2)
	assert.Empty(errList)

	blockBytes := new(bytes.Buffer)
	err := message.MarshalBlock(blockBytes, blk)
	assert.Nil(err)

	// Stupid channels take a while to send something
	time.Sleep(1000 * time.Millisecond)

	// Now, add the hash to validHashes
	score := message.MockScore(hdr, blk.Header.Hash, blockBytes.Bytes())
	vchMsg := message.New(topics.ValidCandidateHash, score)
	errList = eb.Publish(topics.ValidCandidateHash, vchMsg)
	assert.Empty(errList)

	// Now filter the queue
	msg3 := message.New(topics.BestScore, nil)
	errList = eb.Publish(topics.BestScore, msg3)
	assert.Empty(errList)

	// Broker should now be able to provide us with `blk`
	resp, err := rb.Call(topics.GetCandidate, rpcbus.NewRequest(*bytes.NewBuffer(blk.Header.Hash)), 5*time.Second)
	assert.NoError(err)
	cm = resp.(message.Candidate)

	assert.True(blk.Equals(cm.Block))

	// When requesting blk2, we should get an error.
	_, err = rb.Call(topics.GetCandidate, rpcbus.NewRequest(*bytes.NewBuffer(blk2.Header.Hash)), 5*time.Second)
	assert.Equal(candidate.ErrGetCandidateTimeout, err)
}
