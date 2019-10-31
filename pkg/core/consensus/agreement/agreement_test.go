package agreement_test

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
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

	res := <-hlp.CertificateChan
	winningHash := make([]byte, 32)
	if err := encoding.Read256(&res, winningHash); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, hash, winningHash)
}

func wireAgreement(nrProvisioners int) (*consensus.Coordinator, *agreement.Helper) {
	eb := eventbus.New()
	h := agreement.NewHelper(eb, nrProvisioners)
	factory := agreement.NewFactory(eb, h.Keys[0])
	coordinator := consensus.Start(eb, h.Keys[0], factory)
	// starting up the coordinator
	ru := *consensus.MockRoundUpdateBuffer(1, h.P, nil)
	if err := coordinator.CollectRoundUpdate(ru); err != nil {
		panic(err)
	}
	// we need to remove annoying ED25519 verification or the Republisher
	eb.RemoveAllProcessors()
	// Play to step 3, as agreements can only be made on step 3 or later
	// This prevents the mocked events from getting queued
	coordinator.Play()
	coordinator.Play()
	return coordinator, h
}
