package consensus_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
)

func TestThresholdCheck(t *testing.T) {
	threshold := consensus.NewThreshold()
	score := []byte{120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120,
		120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120,
		120, 120, 120, 120, 120, 120}

	assert.False(t, threshold.Exceeds(score))
}

func TestLowerThreshold(t *testing.T) {
	threshold := consensus.NewThreshold()
	score := []byte{120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120,
		120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120,
		120, 120, 120, 120, 120, 120}

	// lower threshold now (it should be divided by 2)
	threshold.Lower()
	assert.True(t, threshold.Exceeds(score))
}

func TestResetThreshold(t *testing.T) {
	threshold := consensus.NewThreshold()
	score := []byte{120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120,
		120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120,
		120, 120, 120, 120, 120, 120}

	// lower threshold now (it should be divided by 2)
	threshold.Lower()
	// then, reset
	threshold.Reset()

	assert.False(t, threshold.Exceeds(score))
}
