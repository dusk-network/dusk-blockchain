package agreement

import (
	"testing"

	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestAccumulatorProcessing(t *testing.T) {
	nr := 50
	hlp := NewHelper(nr)
	hash, _ := crypto.RandEntropy(32)
	handler := NewHandler(hlp.Keys, *hlp.P)
	accumulator := newAccumulator(handler, 4)

	evs := hlp.Spawn(hash)
	for _, msg := range evs {
		accumulator.Process(msg)
	}

	accumulatedAggros := <-accumulator.CollectedVotesChan
	assert.Equal(t, 38, len(accumulatedAggros))
}
