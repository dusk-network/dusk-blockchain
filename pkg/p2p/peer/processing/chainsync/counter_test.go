package chainsync

import (
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"
)

// The timer on the Counter should not fire after the sync completes, to avoid it from
// messing up an ongoing sync initiated afterwards.
func TestStopTimerGoroutine(t *testing.T) {
	assert := assert.New(t)
	c := NewCounter()

	// Set syncTime to something more reasonable for a unit test
	syncTime = 1 * time.Second

	c.StartSyncing(1, "test_peer_addr")

	c.Decrement()

	// Set syncTime back to original value, so we can easily check the effects of the previous timer
	syncTime = 30 * time.Second
	c.StartSyncing(1, "test_peer_addr")

	// Wait one second, and see if the old timer fires
	time.Sleep(1 * time.Second)
	assert.True(c.IsSyncing())
}
