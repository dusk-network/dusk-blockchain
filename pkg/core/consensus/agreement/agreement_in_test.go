package agreement

import (
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestCollectAgreementEvent(t *testing.T) {
	eb := eventbus.New()
	hlp, hash := ProduceWinningHash(eb, 3)

	winningHash := <-hlp.WinningHashChan
	assert.Equal(t, hash, winningHash.Bytes())
}

func TestFinalize(t *testing.T) {
	eb := eventbus.New()
	hlp, _ := ProduceWinningHash(eb, 3)

	<-hlp.WinningHashChan
	hlp.Aggro.Finalize()
	hash, _ := crypto.RandEntropy(32)
	hlp.Spawn(hash)

	select {
	case <-hlp.WinningHashChan:
		assert.FailNow(t, "there should be no activity on a finalized component")
	case <-time.After(100 * time.Millisecond):
		//all good
	}
}
