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

	cb := func(ctx context.Context) (bool, error) {
		// avoiding spinning too fast
		t := time.After(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			return true, nil
		case <-t:
			return false, nil
		}
	}

	l := loop.New(e, consensus.MockPhase(cb), agreement.New(e))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// the cancelation after 100ms should make the agreement end its loop with
	// no error
	require.NoError(t, l.Spin(ctx, consensus.RoundUpdate{Round: uint64(1)}))
}
