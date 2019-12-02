package agreement_test

import (
	"runtime"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	logrus.SetLevel(logrus.TraceLevel)
}

// Test the accumulation of agreement events. It should result in the agreement component
// publishing a round update.
func TestAgreement(t *testing.T) {
	nr := 50
	_, hlp := wireAgreement(nr)
	hash, _ := crypto.RandEntropy(32)
	for i := 0; i < nr; i++ {
		aev := agreement.MockWire(hash, 1, 3, hlp.Keys, hlp.P, i)
		hlp.Bus.Publish(topics.Agreement, aev)
	}

	res := <-hlp.FinalizeChan
	// discard round
	var round uint64
	if err := encoding.ReadUint64LE(&res, &round); err != nil {
		t.Fatal(err)
	}

	winningHash := make([]byte, 32)
	if err := encoding.Read256(&res, winningHash); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, hash, winningHash)
}

// Test that we properly clean up after calling Finalize.
func TestFinalize(t *testing.T) {
	numGRBefore := runtime.NumGoroutine()
	// Create a set of 100 agreement components, and finalize them immediately
	for i := 0; i < 100; i++ {
		c, _ := wireAgreement(50)
		c.FinalizeRound()
	}

	// Ensure we have freed up all of the resources associated with these components
	numGRAfter := runtime.NumGoroutine()
	// We should have roughly the same amount of goroutines
	assert.InDelta(t, numGRBefore, numGRAfter, 10.0)
}

func wireAgreement(nrProvisioners int) (*consensus.Coordinator, *agreement.Helper) {
	eb, rpc := eventbus.New(), rpcbus.New()
	h := agreement.NewHelper(eb, nrProvisioners)
	factory := agreement.NewFactory(eb, h.Keys[0])
	coordinator := consensus.Start(eb, rpc, h.Keys[0], factory)
	// starting up the coordinator
	ru := *consensus.MockRoundUpdateBuffer(1, h.P, nil)
	if err := coordinator.CollectRoundUpdate(ru); err != nil {
		panic(err)
	}
	// we need to remove annoying ED25519 verification or the Republisher
	eb.RemoveAllProcessors()
	// Play to step 3, as agreements can only be made on step 3 or later
	// This prevents the mocked events from getting queued
	coordinator.Play(h.Aggro.ID())
	coordinator.Play(h.Aggro.ID())
	return coordinator, h
}
