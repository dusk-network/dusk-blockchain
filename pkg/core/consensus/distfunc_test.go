package consensus_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestPDF(t *testing.T) {
	// Add Fixed Test vectors
	for i := 0; i < 100; i++ {

		Y, _ := crypto.RandEntropy(32)

		res, err := consensus.GenerateScore(22000, Y)
		assert.NotEqual(t, uint64(0), res)
		assert.Equal(t, nil, err)

	}

}
func TestWrongYLen(t *testing.T) {

	Y, _ := crypto.RandEntropy(33)

	res, err := consensus.GenerateScore(22000, Y)
	assert.NotEqual(t, nil, err)
	assert.Equal(t, uint64(0), res)

}
