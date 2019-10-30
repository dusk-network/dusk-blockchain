package firststep

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/stretchr/testify/assert"
)

// TestSuccessfulAggro tests that upon collection of a quorum of events, a valid StepVotes get produced
func TestSuccessfulAggro(t *testing.T) {
	eb, rbus := eventbus.New(), rpcbus.New()
	hlp, hash := Kickstart(eb, rbus, 10)
	evs := hlp.Spawn(hash)

	res := make(chan error, 1)
	test := func(hash []byte, svs ...*agreement.StepVotes) {
		assert.Equal(t, hlp.Step(), svs[0].Step)
		res <- hlp.Verify(hash, svs[0], 2)
	}

	aggregator := newAggregator(test, hlp.Handler, hlp.RBus)

	for _, ev := range evs {
		r := reduction.Reduction{}
		_ = reduction.Unmarshal(&ev.Payload, &r)
		if !assert.NoError(t, aggregator.collectVote(r, ev.Header)) {
			assert.FailNow(t, "error in collecting votes")
		}
	}

	err := <-res
	assert.NoError(t, err)
}

// TestInvalidBlock tests that upon collection of a quorum of events, a valid StepVotes get produced
func TestInvalidBlock(t *testing.T) {
	eb, rbus := eventbus.New(), rpcbus.New()
	hlp, hash := Kickstart(eb, rbus, 10)
	hlp.FailOnVerification(true)
	evs := hlp.Spawn(hash)

	res := make(chan struct{}, 1)
	test := func(hash []byte, svs ...*agreement.StepVotes) {
		assert.Equal(t, emptyHash[:], hash)
		assert.Equal(t, 0, len(svs))
		res <- struct{}{}
	}

	aggregator := newAggregator(test, hlp.Handler, hlp.RBus)

	for _, ev := range evs {
		r := reduction.Reduction{}
		_ = reduction.Unmarshal(&ev.Payload, &r)
		if !assert.NoError(t, aggregator.collectVote(r, ev.Header)) {
			assert.FailNow(t, "error in collecting votes")
		}
	}

	<-res
}
