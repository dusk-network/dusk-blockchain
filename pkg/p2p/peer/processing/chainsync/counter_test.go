package chainsync

import (
	"bytes"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/stretchr/testify/assert"
)

// The timer on the Counter should not fire after the sync completes, to avoid it from
// messing up an ongoing sync initiated afterwards.
func TestStopTimerGoroutine(t *testing.T) {
	bus := eventbus.New()
	c := NewCounter(bus)

	// Set syncTime to something more reasonable for a unit test
	syncTime = 1 * time.Second

	c.startSyncing(1)

	// Decrement to 0. This should stop the running `listenForTimer` goroutine
	// that's related to the current sync session.
	bus.Publish(string(topics.AcceptedBlock), bytes.Buffer{})

	// Set syncTime back to original value, so we can easily check the effects of the previous timer
	syncTime = 30 * time.Second
	c.startSyncing(1)

	// Wait one second, and see if the old timer fires
	time.Sleep(1 * time.Second)
	assert.True(t, c.IsSyncing())
}
