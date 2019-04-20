package selection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThresholdCheck(t *testing.T) {
	handler := newScoreHandler()

	// Make score event with a score that is below threshold
	ev := &ScoreEvent{
		Score: []byte{120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120,
			120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120,
			120, 120, 120, 120, 120, 120},
	}

	err := handler.Verify(ev)
	assert.Equal(t, errScoreBelowThreshold, err)
}
