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

func TestLowerThreshold(t *testing.T) {
	handler := newScoreHandler()
	// Make score event with a score that is below threshold
	ev := &ScoreEvent{
		Score: []byte{120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120,
			120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120,
			120, 120, 120, 120, 120, 120},
	}

	// lower threshold now (it should be divided by 2)
	handler.LowerThreshold()
	err := handler.checkThreshold(ev.Score)
	assert.Nil(t, err)
}

func TestResetThreshold(t *testing.T) {
	handler := newScoreHandler()
	// Make score event with a score that is below threshold
	ev := &ScoreEvent{
		Score: []byte{120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120,
			120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120, 120,
			120, 120, 120, 120, 120, 120},
	}

	// lower threshold now (it should be divided by 2)
	handler.LowerThreshold()
	// then, reset
	handler.ResetThreshold()

	err := handler.Verify(ev)
	assert.Equal(t, errScoreBelowThreshold, err)
}
