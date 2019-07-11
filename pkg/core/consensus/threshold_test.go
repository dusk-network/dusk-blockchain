package consensus_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
)

var score = []byte{120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120,
	120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120,
	120, 120, 120, 120, 120, 120}

func TestThresholdCheck(t *testing.T) {
	threshold := consensus.NewThreshold()
	// The standard Threshold is set to [170, 170, ...], so `score` should not exceed it.
	assert.False(t, threshold.Exceeds(score))
}

func TestLowerThreshold(t *testing.T) {
	threshold := consensus.NewThreshold()
	// lower threshold now (it should be divided by 2, bringing it to [85, 85, ...])
	threshold.Lower()
	// `score` should now exceed the threshold
	assert.True(t, threshold.Exceeds(score))
}

func TestResetThreshold(t *testing.T) {
	threshold := consensus.NewThreshold()

	threshold.Lower()
	threshold.Reset()

	assert.False(t, threshold.Exceeds(score))
}
