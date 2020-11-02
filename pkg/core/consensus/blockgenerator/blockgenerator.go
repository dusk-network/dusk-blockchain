package blockgenerator

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/blockgenerator/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/blockgenerator/score"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
)

type scoreGenerator = score.Generator
type candidateGenerator = candidate.Generator

type blockGenerator struct {
	scoreGenerator
	candidateGenerator
}

// BlockGenerator ...
type BlockGenerator interface {
	scoreGenerator
	candidateGenerator
}

// New creates a new BlockGenerator
func New(e *consensus.Emitter, genPubKey *keys.PublicKey, db database.DB) (BlockGenerator, error) {
	s, err := score.New(e, db)
	if err != nil {
		return nil, err
	}
	c := candidate.New(e, genPubKey)

	return &blockGenerator{
		scoreGenerator:     s,
		candidateGenerator: c,
	}, nil
}

// Mock ...
func Mock(e *consensus.Emitter) BlockGenerator {
	return &blockGenerator{
		scoreGenerator:     score.Mock(e),
		candidateGenerator: candidate.Mock(e),
	}
}
