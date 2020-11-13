package candidate_test

import (
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

func TestPublisher(t *testing.T) {
	bus := eventbus.New()
	b := candidate.NewBroker(bus)

	// Catch incoming candidates
	candidateChan := make(chan message.Message, 1)
	cChan := eventbus.NewChanListener(candidateChan)
	bus.Subscribe(topics.Candidate, cChan)

	cm := mockCandidate()

	b.Process(message.New(topics.Candidate, cm))

	// Should now have something on the candidateChan
	select {
	case <-candidateChan:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("did not receive a candidate message")
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
