package reduction

import (
	"bytes"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

// TestReductionIntegrity tests *synchronously* the full chain of a Reduction
// message, from creation (by the reducer) to unpacking (supposedly from the
// peer). The EventBuffer is skipped (while the same processing is applied) to
// avoid the concurrent process that it would otherwise introduce
func TestReductionIntegrity(t *testing.T) {
	k, _ := user.NewRandKeys()

	// Sending a mock reduction event out
	blockHash, _ := crypto.RandEntropy(32)
	broker := initBroker(eventbus.New(), k, time.Second, nil)
	broker.propagateRound(consensus.MockRoundUpdate(1, nil, nil))
	vote, err := broker.Reducer.GenerateReduction(blockHash)
	if err != nil {
		assert.FailNow(t, "generate reduction returned error")
	}

	topic, _ := topics.Extract(vote)
	assert.Equal(t, topics.Reduction, topic)

	gossip := processing.NewGossip(protocol.DevNet)
	if !assert.NoError(t, gossip.Process(vote)) {
		assert.FailNow(t, "Gossip preprocessor returned error")
	}

	length, err := gossip.UnpackLength(vote)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "Gossip unpacking returned error")
	}

	packet := make([]byte, length)
	if read, err := vote.Read(packet); err != nil || uint64(read) != length {
		assert.FailNowf(t, "could not read packet of %s bytes after frame", string(length))
	}

	// Receiving a mock reduction event in
	buf := bytes.NewBuffer(packet)
	v := &consensus.Validator{}
	assert.NoError(t, v.Process(buf))
}
