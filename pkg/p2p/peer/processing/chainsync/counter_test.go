package chainsync

import (
	"bytes"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	assert "github.com/stretchr/testify/require"
)

// The timer on the Counter should not fire after the sync completes, to avoid it from
// messing up an ongoing sync initiated afterwards.
func TestStopTimerGoroutine(t *testing.T) {
	assert := assert.New(t)
	bus := rpcbus.New()
	c, err := NewCounter(bus)
	assert.NoError(err)

	// Set syncTime to something more reasonable for a unit test
	syncTime = 1 * time.Second

	c.StartSyncing(1)

	//msg := message.New(topics.AcceptedBlock, message.SafeBuffer{})
	//// Decrement to 0. This should stop the running `listenForTimer` goroutine
	//// that's related to the current sync session.
	//bus.Publish(topics.AcceptedBlock, msg)
	//// making sure that the message got propagated
	//time.Sleep(100 * time.Millisecond)

	params := new(bytes.Buffer)
	resp, err := c.bus.Call(topics.AcceptedBlock, rpcbus.NewRequest(*params), time.Second) //20 is tmp value for further checks

	// testing that there is no error and an empty response (counter.decrement
	// does not return anything)
	assert.NoError(err)
	assert.NotNil(resp)

	// Set syncTime back to original value, so we can easily check the effects of the previous timer
	syncTime = 30 * time.Second
	c.StartSyncing(1)

	// Wait one second, and see if the old timer fires
	time.Sleep(1 * time.Second)
	assert.True(c.IsSyncing())
}
