package secondstep

import (
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/stretchr/testify/assert"
)

func TestSuccessfulAggro(t *testing.T) {
	hlp, hash := Kickstart(10, 1*time.Second)
	evs := hlp.Spawn(hash)

	res := make(chan error, 1)
	test := func(hash []byte, svs ...*message.StepVotes) {
		res <- hlp.Verify(hash, *svs[1], hlp.Step())
	}

	var sv *message.StepVotes
	aggregator := newAggregator(test, hlp.Handler, sv)

	go func() {
		for _, r := range evs {
			if !assert.NoError(t, aggregator.collectVote(r)) {
				assert.FailNow(t, "error in collecting votes")
			}
		}
	}()

	err := <-res
	assert.NoError(t, err)
}
