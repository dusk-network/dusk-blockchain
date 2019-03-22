package selection

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
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

	se := &ScoreEvent{
		Round:         uint64(23),
		Step:          uint8(2),
		Score:         score,
		Proof:         proof,
		Z:             z,
		Seed:          seed,
		BidListSubset: bidListSubset,
		CandidateHash: candidateHash,
	}
	validate := func(buf *bytes.Buffer) error { return nil }
	unMarshaller := NewUnMarshaller(validate)

	bin := make([]byte, 3000)
	buf := bytes.NewBuffer(bin)
	assert.NoError(t, unMarshaller.Marshal(buf, se))

	other := &ScoreEvent{}
	assert.NoError(t, unMarshaller.Unmarshal(buf, other))
	assert.Equal(t, se, other)
}
