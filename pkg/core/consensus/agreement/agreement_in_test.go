package agreement

import (
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestCollectAgreementEvent(t *testing.T) {
	eb := eventbus.New()
	hlp, hash := ProduceWinningHash(eb, 3)

	certificateBuf := <-hlp.FinalizeChan
	// read off the round
	var round uint64
	if err := encoding.ReadUint64LE(&certificateBuf, &round); err != nil {
		t.Fatal(err)
	}

	certHash := make([]byte, 32)
	if err := encoding.Read256(&certificateBuf, certHash); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, hash, certHash)
}

func TestFinalize(t *testing.T) {
	eb := eventbus.New()
	hlp, _ := ProduceWinningHash(eb, 3)

	<-hlp.FinalizeChan
	hlp.Aggro.Finalize()
	hash, _ := crypto.RandEntropy(32)
	hlp.Spawn(hash)

	select {
	case <-hlp.FinalizeChan:
		assert.FailNow(t, "there should be no activity on a finalized component")
	case <-time.After(100 * time.Millisecond):
		//all good
	}
}
