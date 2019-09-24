package generation_test

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/generation"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-wallet/key"
	zkproof "github.com/dusk-network/dusk-zkproof"
	"github.com/stretchr/testify/assert"
)

// Test that all of the functionality around score/proof generation works as intended.
// Note that the proof generator is mocked here, so the actual validity of the data
// is not tested.
func TestScoreGeneration(t *testing.T) {
	d := ristretto.Point{}
	d.Rand()
	k := ristretto.Scalar{}
	k.Rand()

	keys, _ := user.NewRandKeys()
	publicKey := key.NewKeyPair(nil).PublicKey()
	eb, streamer := helper.CreateGossipStreamer()

	gen := &mockGenerator{t}

	// create lite db
	_, db := lite.CreateDBConnection()
	// Add a genesis block so we don't run into any panics
	blk := helper.RandomBlock(t, 0, 2)
	bid, err := helper.RandomBidTx(t, false)
	if err != nil {
		t.Fatal(err)
	}

	bid.Outputs[0].Commitment = d
	m := zkproof.CalculateM(k)
	bid.M = m.Bytes()

	blk.AddTx(bid)
	err = db.Update(func(t database.Transaction) error {
		return t.StoreBlock(blk)
	})
	assert.NoError(t, err)

	// launch score component
	generation.Launch(eb, nil, k, keys, publicKey, gen, gen, db)

	// send a round update to start generation
	eb.Publish(msg.RoundUpdateTopic, *consensus.MockRoundUpdateBuffer(1, nil, nil))

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

func (m *mockGenerator) Generate(roundUpdate consensus.RoundUpdate) (block.Block, selection.ScoreEvent, error) {
	seed, _ := crypto.RandEntropy(33)
	hash, _ := crypto.RandEntropy(32)
	proof := m.GenerateProof(seed, user.BidList{})
	blk, err := m.GenerateBlock(roundUpdate.Round, seed, proof.Proof, proof.Score, hash)
	if err != nil {
		panic(err)
	}

	return *blk, selection.ScoreEvent{
		Round:         roundUpdate.Round,
		Score:         proof.Score,
		Proof:         proof.Proof,
		Z:             hash,
		BidListSubset: hash,
		PrevHash:      hash,
		Seed:          seed,
		VoteHash:      hash,
	}, nil
}

func (m *mockGenerator) GenerateBlock(round uint64, seed, proof, score, prevBlockHash []byte) (*block.Block, error) {
	return helper.RandomBlock(m.t, round, 10), nil
}

func (m *mockGenerator) GenerateProof(seed []byte, bl user.BidList) zkproof.ZkProof {
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

func (m *mockGenerator) InBidList(bidList user.BidList) bool     { return true }
func (m *mockGenerator) UpdateBidList(bl user.Bid)               {}
func (m *mockGenerator) RemoveExpiredBids(round uint64)          {}
func (m *mockGenerator) UpdateProofValues(d, M ristretto.Scalar) {}
