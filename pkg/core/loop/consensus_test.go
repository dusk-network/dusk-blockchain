package loop_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/loop"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
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
	// a nil return value
	cert, hash, comm, err := l.Spin(ctx, consensus.MockPhase(cb), agreement.New(e), consensus.RoundUpdate{Round: uint64(1)})
	require.Nil(t, cert)
	require.Nil(t, hash)
	require.Nil(t, comm)
	require.Nil(t, err)
}

// step is used by TestAgreementCompletion to test that any step would
// properly get canceled
type step struct {
	wg *sync.WaitGroup
}

func (h *step) String() string {
	return "step"
}

func (h *step) Run(ctx context.Context, _ *consensus.Queue, _ chan message.Message, _ consensus.RoundUpdate, _ uint8) consensus.PhaseFn {
	h.wg.Done()
	// if this does not get canceled, the test will timeout
	<-ctx.Done()
	return nil
}

// Fn returns a phase function that simply hangs until canceled
func (h *step) Fn(_ consensus.InternalPacket) consensus.PhaseFn {
	return h
}

// succesfulAgreement is used by TestAgreementCompletion to simulate a
// completing agreement
type succesfulAgreement struct {
	wg *sync.WaitGroup
}

// GetControlFn creates a function that returns after a small sleep. This
// simulates the agreement reaching consensus
func (c *succesfulAgreement) GetControlFn() consensus.ControlFn {
	return func(_ context.Context, _ *consensus.Queue, _ <-chan message.Message, _ consensus.RoundUpdate) (*block.Certificate, []byte, [][]byte) {
		c.wg.Wait()
		return block.EmptyCertificate(), make([]byte, 32), make([][]byte, 0)
	}
}

// TestAgreementCompletion tests that the loop returns with no error when the
// agreement completes normally
func TestAgreementCompletion(t *testing.T) {
	e := consensus.MockEmitter(time.Second, nil)
	ctx := context.Background()
	l := loop.New(e)
	var wg sync.WaitGroup
	wg.Add(1)
	cert, hash, comm, err := l.Spin(ctx, &step{&wg}, &succesfulAgreement{&wg}, consensus.RoundUpdate{Round: uint64(1)})
	require.NotNil(t, cert)
	require.NotNil(t, hash)
	require.NotNil(t, comm)
	require.Nil(t, err)
}

// stallingStep is used by TestStall to test that any step would
// properly get canceled
type stallingStep struct{}

func (h *stallingStep) String() string {
	return "stalling"
}

// Run simply returns a phase function that sleeps and returns itself until canceled
func (h *stallingStep) Run(ctx context.Context, _ *consensus.Queue, _ chan message.Message, _ consensus.RoundUpdate, _ uint8) consensus.PhaseFn {
	time.Sleep(time.Millisecond)
	return h
}

// Fn returns a phase function that simply hangs until canceled
func (h *stallingStep) Fn(_ consensus.InternalPacket) consensus.PhaseFn {
	return h
}

// unsuccesfulAgreement is used by TestStall to simulate an
// agreement that never completes
type unsuccesfulAgreement struct{}

// GetControlFn creates a function that returns after a small sleep. This
// simulates the agreement reaching consensus
func (c *unsuccesfulAgreement) GetControlFn() consensus.ControlFn {
	return func(ctx context.Context, _ *consensus.Queue, _ <-chan message.Message, _ consensus.RoundUpdate) (*block.Certificate, []byte, [][]byte) {
		<-ctx.Done()
		return nil, nil, nil
	}
}

// TestStall tests that the agreement loop gets canceled when the
// state machine reeaches the maximum amount of steps
func TestStall(t *testing.T) {
	e := consensus.MockEmitter(time.Second, nil)
	ctx := context.Background()
	l := loop.New(e)
	_, _, _, _ = l.Spin(ctx, &stallingStep{}, &unsuccesfulAgreement{}, consensus.RoundUpdate{Round: uint64(1)})
	// require.Equal(t, loop.ErrMaxStepsReached, err)
}
