package selection_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestUnMarshal(t *testing.T) {

	// 32 bytes
	score, _ := crypto.RandEntropy(32)
	// Var Bytes
	proof, _ := crypto.RandEntropy(1477)
	// 32 bytes
	z, _ := crypto.RandEntropy(32)
	// Var Bytes
	bidListSubset, _ := crypto.RandEntropy(32)
	// BLS is 33 bytes
	seed, _ := crypto.RandEntropy(33)
	// 32 bytes
	candidateHash, _ := crypto.RandEntropy(32)

	se := selection.ScoreEvent{
		Round:         uint64(23),
		Score:         score,
		Proof:         proof,
		Z:             z,
		Seed:          seed,
		BidListSubset: bidListSubset,
		VoteHash:      candidateHash,
	}

	buf := new(bytes.Buffer)
	assert.NoError(t, selection.MarshalScoreEvent(buf, se))

	other := &selection.ScoreEvent{}
	assert.NoError(t, selection.UnmarshalScoreEvent(buf, other))
	assert.True(t, other.Equal(se))
}
