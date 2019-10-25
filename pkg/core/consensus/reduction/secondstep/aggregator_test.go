package secondstep

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestSuccessfulAggro(t *testing.T) {
	eb, rbus := eventbus.New(), rpcbus.New()
	hlp := NewHelper(eb, rbus, nil, nil, 10)
	hash, _ := crypto.RandEntropy(32)
	evs := hlp.Spawn(hash)

	res := make(chan error, 1)
	test := func(hash []byte, svs ...*agreement.StepVotes) {
		res <- hlp.Verify(hash, svs[1])
	}

	var sv *agreement.StepVotes
	aggregator := newAggregator(test, hlp.Handler, sv)

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
