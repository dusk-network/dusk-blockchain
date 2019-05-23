package generation_test

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/generation"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/zkproof"
)

func TestScoreGeneration(t *testing.T) {
	d := ristretto.Scalar{}
	d.Rand()
	k := ristretto.Scalar{}
	k.Rand()

	eb, streamer := helper.CreateGossipStreamer()

	gen := &mockGenerator{t}

	// launch score component
	generation.Launch(eb, nil, d, k, gen, gen)

	// update the round to start generation
	updateRound(eb, 1)

	buf, err := streamer.Read()
	if err != nil {
		t.Fatal(err)
	}

	sev := &selection.ScoreEvent{}
	if err := selection.UnmarshalScoreEvent(bytes.NewBuffer(buf), sev); err != nil {
		t.Fatal(err)
	}

	// round should be 1
	assert.Equal(t, uint64(1), sev.Round)
}

// mockGenerator is used to test the generation component with the absence
// of the rust blind bid process.
type mockGenerator struct {
	t *testing.T
}

func (m *mockGenerator) GenerateBlock(round uint64, seed []byte) (*block.Block, error) {
	return helper.RandomBlock(m.t, round, 10), nil
}

func (m *mockGenerator) GenerateProof(seed []byte) zkproof.ZkProof {
	proof, _ := crypto.RandEntropy(100)
	// add a score that will always pass threshold
	score, _ := big.NewInt(0).SetString("BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB", 16)
	z, _ := crypto.RandEntropy(32)
	bL, _ := crypto.RandEntropy(32)
	return zkproof.ZkProof{
		Proof:         proof,
		Score:         score.Bytes(),
		Z:             z,
		BinaryBidList: bL,
	}
}

func (m *mockGenerator) UpdateBidList(bl user.BidList) {}

func updateRound(eventBus *wire.EventBus, round uint64) {
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes, round)
	eventBus.Publish(msg.RoundUpdateTopic, bytes.NewBuffer(roundBytes))
}
