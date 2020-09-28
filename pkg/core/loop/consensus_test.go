package loop_test

import (
	"context"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/loop"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/stretchr/testify/require"
)

type mockPhase struct {
	callback func(ctx context.Context) (bool, error)
}

func (m *mockPhase) Fn(_ consensus.InternalPacket) consensus.PhaseFn {
	return m.Run
}

func (m *mockPhase) Run(ctx context.Context, queue *consensus.Queue, evChan chan message.Message, r consensus.RoundUpdate, step uint8) (consensus.PhaseFn, error) {
	if stop, err := m.callback(ctx); err != nil {
		return nil, err
	} else if stop {
		return nil, nil
	}
	return m.Run, nil
}

// TestContextCancellation tests that the loop returns in case of a
// cancellation
func TestContextCancellation(t *testing.T) {
	e := createEmitter(t)

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

	l := loop.New(e, &mockPhase{cb}, agreement.New(e))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// the cancelation after 100ms should make the agreement end its loop with
	// no error
	require.NoError(t, l.Spin(ctx, consensus.RoundUpdate{Round: uint64(1)}))
}

func createEmitter(t *testing.T) *consensus.Emitter {
	keys, err := key.NewRandKeys()
	require.NoError(t, err)

	return &consensus.Emitter{
		EventBus: eventbus.New(),
		Keys:     keys,
	}
}
