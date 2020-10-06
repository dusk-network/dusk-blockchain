package agreement_test

import (
	"context"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockValidity ensures that we don't go into a wild goose chase if our
// mock system gets screwed up
func TestMockValidity(t *testing.T) {
	nr := 50
	hlp := agreement.NewHelper(nr)
	hash, _ := crypto.RandEntropy(32)
	handler := agreement.NewHandler(hlp.Keys, *hlp.P)

	evs := hlp.Spawn(hash)
	for _, ev := range evs {
		if !assert.NoError(t, handler.Verify(ev)) {
			t.FailNow()
		}
	}
}

// Test the accumulation of agreement events. It should result in the agreement component
// sending a valid certificate
func TestAgreement(t *testing.T) {
	nr := 50
	hlp := agreement.NewHelper(nr)
	hash, _ := crypto.RandEntropy(32)

	loop := agreement.New(hlp.Emitter)

	agreementEvs := hlp.Spawn(hash)
	agreementChan := make(chan message.Message, 100)

	for _, aggro := range agreementEvs {
		agreementChan <- message.New(topics.Agreement, aggro)
	}

	ctx := context.Background()
	err := loop.Run(ctx, consensus.NewQueue(), agreementChan, hlp.RoundUpdate(hash))
	require.NoError(t, err)

	res := <-hlp.CertificateChan
	cert := res.Payload().(message.Certificate)
	assert.Equal(t, hash, cert.Ag.State().BlockHash)
}
