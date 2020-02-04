package candidate_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
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

	// Store two blocks in the queue
	blk := helper.RandomBlock(t, 1, 3)
	blk2 := helper.RandomBlock(t, 1, 3)
	cert := block.EmptyCertificate()
	blk.SetHash()
	buf := new(bytes.Buffer)
	if err := marshalling.MarshalBlock(buf, blk); err != nil {
		t.Fatal(err)
	}

	if err := marshalling.MarshalCertificate(buf, cert); err != nil {
		t.Fatal(err)
	}

	buf2 := new(bytes.Buffer)
	if err := marshalling.MarshalBlock(buf2, blk2); err != nil {
		t.Fatal(err)
	}

	if err := marshalling.MarshalCertificate(buf2, cert); err != nil {
		t.Fatal(err)
	}

	eb.Publish(topics.Candidate, buf)
	eb.Publish(topics.Candidate, buf2)
	// Stupid channels take a while to send something
	time.Sleep(1000 * time.Millisecond)

	// Only send a valid hash message for the first one
	eb.Publish(topics.ValidCandidateHash, bytes.NewBuffer(blk.Header.Hash))

	// Now filter the queue
	eb.Publish(topics.BestScore, new(bytes.Buffer))

	// Broker should now be able to provide us with `blk`
	blkBuf, err := rb.Call(rpcbus.GetCandidate, rpcbus.Request{*bytes.NewBuffer(blk.Header.Hash), make(chan rpcbus.Response, 1)}, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	decoded := block.NewBlock()
	if err := marshalling.UnmarshalBlock(&blkBuf, decoded); err != nil {
		t.Fatal(err)
	}

	assert.True(t, blk.Equals(decoded))

	// When requesting blk2, we should get an error.
	_, err = rb.Call(rpcbus.GetCandidate, rpcbus.Request{*bytes.NewBuffer(blk2.Header.Hash), make(chan rpcbus.Response, 1)}, 5*time.Second)
	assert.Equal(t, "request timeout", err.Error())
}
