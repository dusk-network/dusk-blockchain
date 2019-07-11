package selection

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

// MockSelectionEventBuffer mocks a Selection event, marshals it, and returns the
// resulting buffer.
func MockSelectionEventBuffer(round uint64, hash []byte) *bytes.Buffer {
	se := MockSelectionEvent(round, hash)
	r := new(bytes.Buffer)
	_ = MarshalScoreEvent(r, se)
	return r
}

// MockSelectionEvent mocks a Selection event and returns it.
func MockSelectionEvent(round uint64, hash []byte) *ScoreEvent {
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
	se := &ScoreEvent{
		Round:         round,
		Score:         score,
		Proof:         proof,
		Z:             z,
		Seed:          seed,
		BidListSubset: bidListSubset,
		PrevHash:      hash,
		Certificate:   block.EmptyCertificate(),
		VoteHash:      hash,
	}

	return se
}
