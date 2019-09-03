package consensus_test

import (
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/stretchr/testify/assert"
)

func TestTimerDoubling(t *testing.T) {
	timer := consensus.NewTimer(5*time.Second, nil)

	// Double timer 10 times and check timeout
	for i := 0; i < 10; i++ {
		timer.IncreaseTimeOut()
	}

	assert.Equal(t, 5120*time.Second, timer.TimeOut())

	// Double it again, and ensure it stays the same
	timer.IncreaseTimeOut()
	assert.Equal(t, 5120*time.Second, timer.TimeOut())
}
