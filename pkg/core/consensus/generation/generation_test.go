package generation

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"gitlab.dusk.network/dusk-core/zkproof"
)

func TestScoreGeneration(t *testing.T) {
	eb := wire.NewEventBus()
	d := ristretto.Scalar{}
	d.Rand()
	k := ristretto.Scalar{}
	k.Rand()

	// subscribe to gossip topic
	gossipChan := make(chan *bytes.Buffer, 1)
	eb.Subscribe(string(topics.Gossip), gossipChan)

	// launch score component
	broker := LaunchScoreGenerationComponent(eb, d, k)
	// use mockgenerator
	broker.proofGenerator = &mockGenerator{}
	// update the round to start generation
	updateRound(eb, 1)

	// should get a score event out of the gossip topic
	ev := <-gossipChan
	// check the event
	// discard the topic first
	topicBuffer := make([]byte, 15)
	if _, err := ev.Read(topicBuffer); err != nil {
		t.Fatal(err)
	}
	// now unmarshal the event
	sev := &selection.ScoreEvent{}
	if err := selection.UnmarshalScoreEvent(ev, sev); err != nil {
		t.Fatal(err)
	}

	// round should be 1
	assert.Equal(t, uint64(1), sev.Round)
}

// mockGenerator is used to test the generation component with the absence
// of the rust blind bid process.
type mockGenerator struct {
}

func (m *mockGenerator) generateProof(seed []byte) zkproof.ZkProof {
	fmt.Println("generating proof")
	proof, _ := crypto.RandEntropy(100)
	score, _ := crypto.RandEntropy(32)
	z, _ := crypto.RandEntropy(32)
	bL, _ := crypto.RandEntropy(32)
	return zkproof.ZkProof{
		Proof:         proof,
		Score:         score,
		Z:             z,
		BinaryBidList: bL,
	}
}

func (m *mockGenerator) updateBidList(bl user.BidList) {}

func updateRound(eventBus *wire.EventBus, round uint64) {
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes, round)
	eventBus.Publish(msg.RoundUpdateTopic, bytes.NewBuffer(roundBytes))
}
