package agreement

import (
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestAccumulatorProcessing(t *testing.T) {
	nr := 50
	_, hlp := WireAgreement(nr)
	hash, _ := crypto.RandEntropy(32)
	handler := NewHandler(hlp.Keys[0], *hlp.P)
	accumulator := newAccumulator(handler, 4)

	for i := 0; i < nr; i++ {
		a := message.MockAgreement(hash, 1, 3, hlp.Keys, hlp.P, i)
		accumulator.Process(a)
	}

	accumulatedAggros := <-accumulator.CollectedVotesChan
	assert.Equal(t, 38, len(accumulatedAggros))
}

func TestCollectAgreementEvent(t *testing.T) {
	eb := eventbus.New()
	hlp, hash := ProduceWinningHash(eb, 3)

	certMsg := <-hlp.CertificateChan
	aggro := certMsg.Payload().(message.Agreement)

	assert.Equal(t, hash, aggro.State().BlockHash)
}

func TestFinalize(t *testing.T) {
	eb := eventbus.New()
	hlp, _ := ProduceWinningHash(eb, 3)

	<-hlp.CertificateChan
	hlp.Aggro.Finalize()
	hash, _ := crypto.RandEntropy(32)
	hlp.Spawn(hash)

	select {
	case <-hlp.CertificateChan:
		assert.FailNow(t, "there should be no activity on a finalized component")
	case <-time.After(100 * time.Millisecond):
		//all good
	}
}
