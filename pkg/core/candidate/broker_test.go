package candidate_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/block"
	"github.com/stretchr/testify/assert"
)

// Ensures that candidate blocks are only let through if
// a `ValidCandidateHash` message for that block was seen.
func TestValidHashes(t *testing.T) {
	eb, rb := eventbus.New(), rpcbus.New()
	b := candidate.NewBroker(eb, rb)
	go b.Listen()

	blk := helper.RandomBlock(t, 1, 3)
	cert := block.EmptyCertificate()
	blk.SetHash()

	// mocking a header with consistent hash with the block
	hdr := header.Mock()
	hdr.BlockHash = blk.Header.Hash

	// First, attempt to store it without a `ValidCandidateHash` message.
	cm := message.MakeCandidate(blk, cert)
	msg := message.New(topics.Candidate, cm)
	eb.Publish(topics.Candidate, msg)

	// Stupid channels take a while to send something
	time.Sleep(1000 * time.Millisecond)

	// When requesting it, we should get an error.
	_, err := rb.Call(rpcbus.GetCandidate, rpcbus.Request{*bytes.NewBuffer(blk.Header.Hash), make(chan rpcbus.Response, 1)}, 5*time.Second)
	assert.Equal(t, "request timeout", err.Error())

	// Now, add the hash to validHashes
	score := message.MockScore(hdr, blk.Header.Hash)
	vchMsg := message.New(topics.ValidCandidateHash, score)
	eb.Publish(topics.ValidCandidateHash, vchMsg)

	// And try again.
	eb.Publish(topics.Candidate, msg)

	blkBuf, err := rb.Call(rpcbus.GetCandidate, rpcbus.Request{*bytes.NewBuffer(blk.Header.Hash), make(chan rpcbus.Response, 1)}, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	decoded := block.NewBlock()
	if err := message.UnmarshalBlock(&blkBuf, decoded); err != nil {
		t.Fatal(err)
	}

	assert.True(t, blk.Equals(decoded))
}
