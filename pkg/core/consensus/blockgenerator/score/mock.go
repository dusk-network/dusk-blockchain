package score

import (
	"context"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

type mock struct {
	*consensus.Emitter
}

func (m *mock) Generate(ctx context.Context, r consensus.RoundUpdate, step uint8) message.ScoreProposal {
	hash, _ := crypto.RandEntropy(32)

	hdr := header.Header{
		Round:     r.Round,
		Step:      step,
		BlockHash: hash,
		PubKeyBLS: m.Keys.BLSPubKeyBytes,
	}

	return message.MockScoreProposal(hdr)
}

// Mock a score generator
func Mock(e *consensus.Emitter) Generator {
	return &mock{e}
}
