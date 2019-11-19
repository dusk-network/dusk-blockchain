package secondstep

import (
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/stretchr/testify/assert"
)

func TestSuccessfulAggro(t *testing.T) {
	hlp, hash := Kickstart(10, 1*time.Second)
	evs := hlp.Spawn(hash)

	res := make(chan error, 1)
	test := func(hash []byte, svs ...*agreement.StepVotes) {
		res <- hlp.Verify(hash, svs[1], hlp.Step())
	}

	var sv *agreement.StepVotes
	aggregator := newAggregator(test, hlp.Handler, sv)

	go func() {
		for _, ev := range evs {
			r := reduction.Reduction{}
			_ = reduction.Unmarshal(&ev.Payload, &r)
			if !assert.NoError(t, aggregator.collectVote(r, ev.Header)) {
				assert.FailNow(t, "error in collecting votes")
			}
		}
	}()

	err := <-res
	assert.NoError(t, err)
}
