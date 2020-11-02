package candidate

import (
	"context"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

/*
// MockBlock ...
func MockBlock(round uint64, e *consensus.Emitter) *block.Block {
	key := keys.NewPublicKey()

	g := &generator{
		Emitter:   e,
		genPubKey: key,
	}

	// TODO: do we need to generate correct proof and score
	seed, _ := crypto.RandEntropy(33)
	proof, _ := crypto.RandEntropy(32)
	score, _ := crypto.RandEntropy(32)

	b, err := g.GenerateBlock(round, seed, proof, score, make([]byte, 32), [][]byte{{0}})
	if err != nil {
		panic(err)
	}

	return b
}
*/

// MockCandidate ...
func (m *mock) MockCandidate(sev message.ScoreProposal, previousBlock []byte) message.Candidate {
	if previousBlock == nil {
		previousBlock, _ = crypto.RandEntropy(32)
	}

	hdr := sev.State()
	b, err := m.GenerateBlock(hdr.Round, sev.Seed, sev.Proof, sev.Score, previousBlock, [][]byte{hdr.PubKeyBLS})
	if err != nil {
		panic(err)
	}

	return message.Candidate{Block: b, Certificate: block.EmptyCertificate()}
}

type mock struct {
	*generator
}

func (m *mock) GenerateCandidateMessage(ctx context.Context, sev message.ScoreProposal, r consensus.RoundUpdate, step uint8) (*message.Score, error) {
	mockScore := message.MockScore(sev.State(), m.MockCandidate(sev, nil))
	return &mockScore, nil
}

// Mock the candidate generator
func Mock(e *consensus.Emitter) Generator {
	key := keys.NewPublicKey()
	return &mock{
		generator: New(e, key).(*generator),
	}
}
