package loop_test

import (
	"context"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/loop"
	"github.com/stretchr/testify/require"
)

// TestContextCancellation tests that the loop returns in case of a
// cancellation
func TestContextCancellation(t *testing.T) {
	e := consensus.MockEmitter(time.Second, nil)

	cb := func(ctx context.Context) bool {
		// avoiding spinning too fast
		t := time.After(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			return true
		case <-t:
			return false
		}
	}

	l := loop.New(e)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// the cancelation after 100ms should make the agreement end its loop with
	// no error
	require.NoError(t, l.Spin(ctx, consensus.MockPhase(cb), agreement.New(e), consensus.RoundUpdate{Round: uint64(1)}))
}
