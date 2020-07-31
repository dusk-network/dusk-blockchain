package reduction_test

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/stretchr/testify/assert"
)

// Ensure that stopping a timer which was never started does not result
// in a panic.
func TestStopNilTimer(t *testing.T) {
	c := make(chan reduction.HaltMsg, 1)
	timer := reduction.NewTimer(c)
	assert.NotPanics(t, timer.Stop)
}
