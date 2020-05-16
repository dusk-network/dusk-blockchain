package agreement_test

import (
	"runtime"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

// TestMockValidity ensures that we don't go into a wild goose chase if our
// mock system gets screwed up
func TestMockValidity(t *testing.T) {
	nr := 50
	_, hlp := agreement.WireAgreement(nr)
	hash, _ := crypto.RandEntropy(32)
	handler := agreement.NewHandler(hlp.Keys[0], *hlp.P)

	for i := 0; i < nr; i++ {
		a := message.MockAgreement(hash, 1, 3, hlp.Keys, hlp.P, i)
		if !assert.NoError(t, handler.Verify(a)) {
			t.FailNow()
		}
	}
}

// Test the accumulation of agreement events. It should result in the agreement component
// publishing a round update.
// TODO: trap eventual errors
func TestAgreement(t *testing.T) {
	nr := 50
	_, hlp := agreement.WireAgreement(nr)
	hash, _ := crypto.RandEntropy(32)
	for i := 0; i < nr; i++ {
		a := message.MockAgreement(hash, 1, 3, hlp.Keys, hlp.P, i)
		msg := message.New(topics.Agreement, a)
		// marshaling will populate the cache and prevent race-conditions
		// within the Republisher
		message.Marshal(msg)

		hlp.Bus.Publish(topics.Agreement, msg)
	}

	res := <-hlp.CertificateChan
	cert := res.Payload().(message.Certificate)
	assert.Equal(t, hash, cert.Ag.State().BlockHash)
}

// Test that we properly clean up after calling Finalize.
// TODO: trap eventual errors
func TestFinalize(t *testing.T) {
	numGRBefore := runtime.NumGoroutine()
	// Create a set of 100 agreement components, and finalize them immediately
	for i := 0; i < 100; i++ {
		c, _ := agreement.WireAgreement(50)
		c.FinalizeRound()
	}

	// Ensure we have freed up all of the resources associated with these components
	numGRAfter := runtime.NumGoroutine()
	// We should have roughly the same amount of goroutines
	assert.InDelta(t, numGRBefore, numGRAfter, 10.0)
}
