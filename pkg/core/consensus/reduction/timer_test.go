package reduction_test

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/stretchr/testify/assert"
)

// Ensure that stopping a timer which was never started does not result
// in a panic.
func TestStopNilTimer(t *testing.T) {
	timer := reduction.NewTimer(func([]byte, ...*message.StepVotes) {})
	assert.NotPanics(t, timer.Stop)
}
