package secondstep

/*
import (
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/stretchr/testify/assert"
)

func TestSuccessfulAggro(t *testing.T) {
	hlp, hash := Kickstart(10, 1*time.Second)
	evs := hlp.Spawn(hash)

	res := make(chan reduction.HaltMsg, 1)

	var sv *message.StepVotes
	aggregator := newAggregator(res, hlp.Handler, sv)

	go func() {
		for _, r := range evs {
			if !assert.NoError(t, aggregator.collectVote(r)) {
				assert.FailNow(t, "error in collecting votes")
			}
		}
	}()

	m := <-res
	assert.NoError(t, hlp.Verify(m.Hash, *m.Sv[1], hlp.Step()))
}
*/
